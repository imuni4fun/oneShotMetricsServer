setup:
	# GOPRIVATE=github.com/imuni4fun/fadingMetricsCache go get github.com/imuni4fun/fadingMetricsCache
	go get goyave.dev/goyave/v5@v5.4.0

build: setup
	go build .

build-docker:
	docker build -t imuni4fun/oneShotMetricsServer:$(git describe) .

build-docker-verbose:
	docker build -t imuni4fun/oneShotMetricsServer:$(git describe) --no-cache --progress=plain .

test: setup
	go test . -count 1 -v

run: setup
	go run .