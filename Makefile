GIT_TAG:=$(shell git describe --tags --dirty --always)
GHCR:=ghcr.io/imuni4fun/one-shot-metrics-server

setup:
	go get goyave.dev/goyave/v5@v5.4.0
	go get github.com/imuni4fun/fadingMetricsCache@v0.0.4
	
build: setup
	go build .

build-docker:
	docker build -t $(GHCR):$(GIT_TAG) .

build-docker-verbose:
	docker build -t $(GHCR):$(GIT_TAG) --no-cache --progress=plain .

push-docker: build-docker
	cat ~/.github/tokens/oneShotMetricsServer | docker login ghcr.io -u USERNAME --password-stdin
	docker push $(GHCR):$(GIT_TAG)

test: setup
	go test . -count 1 -v

run: setup
	go run .