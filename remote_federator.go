package main

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type Storage struct {
	storage map[string]*model.Sample
	mu      *sync.RWMutex
}

func NewStorage() Storage {
	return Storage{
		storage: map[string]*model.Sample{},
		mu:      &sync.RWMutex{},
	}
}

func (s Storage) Write(samples model.Samples) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sample := range samples {
		s.storage[sample.Metric.String()] = sample
	}
	return nil
}

func (s Storage) Read(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.storage {
		if _, err := fmt.Fprintln(w, v.Metric.String(), v.Value.String(), v.Timestamp.String()); err != nil {
			return err
		}
	}
	return nil
}

func timeseries2Samples(ts prompb.TimeSeries) model.Samples {
	var samples model.Samples
	m := make(model.Metric, len(ts.Labels))
	for _, l := range ts.Labels {
		m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
	}
	for _, s := range ts.Samples {
		samples = append(samples, &model.Sample{
			Metric:    m,
			Value:     model.SampleValue(s.Value),
			Timestamp: model.Time(s.Timestamp),
		})
	}
	return samples
}

func main() {
	storage := NewStorage()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Remote Federator</title></head>
             <body>
             <h1>Remote Federator</h1>
             <p><a href='` + "/metrics" + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, ts := range req.Timeseries {
			if err := storage.Write(timeseries2Samples(ts)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				os.Exit(0)
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/federate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		if err := storage.Read(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(":9999", nil))
}
