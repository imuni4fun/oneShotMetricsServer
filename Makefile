GIT_TAG:=$(shell git describe --tags)

setup:
	go get goyave.dev/goyave/v5@v5.4.0
	go get github.com/imuni4fun/fadingMetricsCache@v0.0.1
	
build: setup
	go build .

build-docker:
	docker build -t imuni4fun/one_shot_metrics_server:$(GIT_TAG) .

build-docker-verbose:
	docker build -t imuni4fun/one_shot_metrics_server:$(GIT_TAG) --no-cache --progress=plain .

push-docker: build-docker
	docker build -t imuni4fun/one_shot_metrics_server:$(GIT_TAG) .

test: setup
	go test . -count 1 -v

run: setup
	go run .