package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/imuni4fun/fadingMetricsCache"
	"github.com/stretchr/testify/assert"
	"goyave.dev/goyave/v5/config"
	"goyave.dev/goyave/v5/util/errors"
)

// description of test
func TestConfigure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runServer(ctx)
	}()
	time.Sleep(1 * time.Second)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.(*errors.Error).String())
		os.Exit(1)
	}

	logInfof("extracted listener port %v for test harness\n", cfg.GetInt("server.port"))
	cache := fadingMetricsCache.FadingMetricsCache{}
	cache.Configure(ctx, time.Second*5, 2, 1000000)
	fmt.Println("created cache and did not crash!")
}

// description of test
func TestPost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runServer(ctx)
	}()
	time.Sleep(1 * time.Second)

	httpPostEvent(map[string]string{
		"type":   "testResult",
		"result": "pass",
	})
	fmt.Println("did not crash!")
}

// description of test
func TestScrape(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runServer(ctx)
	}()
	time.Sleep(1 * time.Second)

	httpGetMetrics() // register scraper
	httpPostEvent(map[string]string{
		"type":   "testResult",
		"result": "pass",
	})
	httpPostEvent(map[string]string{
		"type":   "testResult",
		"result": "fail",
	})
	result := httpGetMetrics()

	found := false
	for _, str := range result {
		if strings.Contains(str, `type="testResult"`) && strings.Contains(str, `result="pass"`) {
			logInfof("found metric: %s", str)
			found = true
			break
		}
	}
	assert.True(t, found, "did not find expected metric")

	found = false
	for _, str := range result {
		if strings.Contains(str, `type="testResult"`) && strings.Contains(str, `result="fail"`) {
			logInfof("found metric: %s", str)
			found = true
			break
		}
	}
	assert.True(t, found, "did not find expected metric")
}

func TestParseScraper(t *testing.T) {
	testStrings := map[string]string{
		"203.0.113.0":                                   "203.0.113.0",
		"10.133.107.114:35282":                          "10.133.107.114",
		"2001:db8:3333:4444:5555:6666:7777:8888":        "2001:db8:3333:4444:5555:6666:7777:8888",
		"[2001:db8:3333:4444:5555:6666:7777:8888]:8080": "2001:db8:3333:4444:5555:6666:7777:8888",
		"[2001:db8:0::1]:80":                            "2001:db8:0::1",
		"2001:db8:0::1":                                 "2001:db8:0::1",
	}
	failStrings := []string{}
	for k, v := range testStrings {
		parsed := getScraperFromIP(k)
		assert.NotEqual(t, "<nil>", parsed, "Parse should not fail for %s", k)
		assert.NotEqual(t, "", parsed, "Parse should not fail for %s", k)
		assert.Equal(t, v, parsed, "Parse for %s should return %s, not %s", k, v, parsed)
		logInfof("address %s --> %s", k, parsed)
	}
	for _, str := range failStrings {
		assert.Panicsf(t, func() {
			_ = getScraperFromIP(str)
		}, "Unsupported address %s should panic", str)
		logInfof("address %s --> should panic", str)
	}
}

func httpPostEvent(content map[string]string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.(*errors.Error).String())
		os.Exit(1)
	}
	port := cfg.GetInt("server.port")
	params := []string{}
	for k, v := range content {
		params = append(params, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	postUrl := fmt.Sprintf("http://localhost:%d/event?%s", port, strings.Join(params, "&"))
	logDebugf("posting to %s\n", postUrl)
	req, err := http.NewRequest(http.MethodPost, postUrl, bytes.NewReader([]byte{}))
	if err != nil {
		logWarnf("client: could not create request: %s\n", err)
		os.Exit(2)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logWarnf("client: error making http request: %s\n", err)
		os.Exit(3)
	}

	logDebugf("client: got response!\n")
	logDebugf("client: status code: %d\n", res.StatusCode)
}

func httpGetMetrics() []string {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.(*errors.Error).String())
		os.Exit(1)
	}
	port := cfg.GetInt("server.port")
	getUrl := fmt.Sprintf("http://localhost:%d/metrics", port)
	logDebugf("posting to %s\n", getUrl)
	req, err := http.NewRequest(http.MethodGet, getUrl, bytes.NewReader([]byte{}))
	if err != nil {
		logWarnf("client: could not create request: %s\n", err)
		os.Exit(2)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logWarnf("client: error making http request: %s\n", err)
		os.Exit(3)
	}

	logDebugf("client: got response!\n")
	logDebugf("client: status code: %d\n", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		logWarnf("client: did not receive StatusOK response code: %d\n", res.StatusCode)
		os.Exit(4)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logWarnf("client: could not read response body: %s\n", err)
		os.Exit(5)
	}

	logDebugf("client: response body:\n%s\n", resBody)

	return strings.Split(string(resBody), "\n")
}
