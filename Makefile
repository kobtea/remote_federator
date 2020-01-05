VERSION    := v$(shell cat VERSION)

build:
	@echo '>> build'
	@CGO_ENABLED=0 go build .

docker-build: build
	@echo '>> build docker image'
	@docker build -t kobtea/remote-federator:$(shell cat VERSION) .
	@docker build -t kobtea/remote-federator:latest .
