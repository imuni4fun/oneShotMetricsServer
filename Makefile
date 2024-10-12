setup:
	GOPRIVATE=github.com/imuni4fun/fadingMetricsCache go get github.com/imuni4fun/fadingMetricsCache
	go get goyave.dev/goyave/v4@v4.4.11

build: setup
	go build .

test: setup
	go test . -count 1 -v

run: setup
	go run .