FROM quay.io/prometheus/busybox:latest
LABEL maintainer="kobtea9696@gmail.com"

COPY remote_federator /bin/remote_federator

EXPOSE 9999
ENTRYPOINT ["/bin/remote_federator"]
