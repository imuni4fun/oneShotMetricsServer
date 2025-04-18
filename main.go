package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/imuni4fun/fadingMetricsCache"
	"goyave.dev/goyave/v5"
	"goyave.dev/goyave/v5/config"
	"goyave.dev/goyave/v5/util/errors"
)

var cache = fadingMetricsCache.FadingMetricsCache{}
var eventCounter atomic.Int64
var useExplicitTimestamps = false

const flagUseExplicitTimestamps = "USE_EXPLICIT_TIMESTAMPS"

func main() {
	switch os.Getenv("LOG_LEVEL") {
	case "ERROR":
		slog.SetLogLoggerLevel(slog.LevelError)
	case "WARN":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "INFO":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "DEBUG":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	default:
	}

	useExplicitTimestamps = strings.ToLower(os.Getenv(flagUseExplicitTimestamps)) == "true"

	runServer(context.Background())
}

func runServer(ctx context.Context) {
	cache.Configure(ctx, 10*time.Minute, 5, 10000)

	logInfof("Using explicit timestamps: %t", useExplicitTimestamps)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.(*errors.Error).String())
		os.Exit(1)
	}

	opts := goyave.Options{
		Config: cfg,
	}

	server, err := goyave.New(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.(*errors.Error).String())
		os.Exit(2)
	}

	server.RegisterSignalHook()

	server.RegisterStartupHook(func(s *goyave.Server) {
		if !s.IsReady() {
			return
		}
	})

	server.RegisterShutdownHook(func(s *goyave.Server) {
		s.Logger.Info("Server is shutting down")
	})

	server.RegisterRoutes(registerRoutes)

	go func() {
		<-ctx.Done()
		server.Stop()
	}()

	server.Logger.Info(fmt.Sprintf("Server is starting to listen on %s:%d", cfg.GetString("server.host"), cfg.GetInt("server.port")))

	if err := server.Start(); err != nil {
		server.Logger.Error(err)
		os.Exit(3)
	}
}

func registerRoutes(server *goyave.Server, router *goyave.Router) {
	router.Get("/", handleGetHome)
	router.Post("/event", handlePostEvent)
	router.Get("/metrics", handleGetMetrics)
	router.Get("/healthz", handleGetHealthz)
}

func handleGetHome(response *goyave.Response, request *goyave.Request) {
	response.String(http.StatusOK, "Welcome to Events-to-metrics!")
}

func handleGetHealthz(response *goyave.Response, request *goyave.Request) {
	response.String(http.StatusOK, "I'm not quite dead!")
}

func handlePostEvent(response *goyave.Response, request *goyave.Request) {

	logDebugf("-------------------\n")
	logDebugf("method : %v\n", request.Method())
	logDebugf("path   : %v\n", request.URL().Path)
	logDebugf("remote : %v\n", request.RemoteAddress())
	if err := request.Request().ParseForm(); err != nil {
		response.String(http.StatusBadRequest, "failed to parse labels: "+err.Error())
	}
	logDebugf("labels : %v\n", request.Request().Form)
	labels := map[string]string{}
	value := 1.0
	for k, v := range request.Request().Form {
		if floatVal, useAsValue := tryParseValueAsFloat(k, v); useAsValue { // interpret as value
			value = floatVal
			logDebugf("value : %v\n", floatVal)
		} else { // interpret as label (no spaces allowed in label name)
			labels[strings.ReplaceAll(k, " ", "_")] = v[0]
			logDebugf("label : %v: %v\n", k, v[0])
		}
	}
	for k, v := range request.Header() {
		logDebugf("header : %s = %s\n", k, v)
	}
	cache.RegisterValue("events_to_metrics", labels, value, useExplicitTimestamps)
	eventCounter.Add(1)
	response.Status(http.StatusOK)
}

func tryParseValueAsFloat(field string, value []string) (float64, bool) {
	if field != "value" || len(value) != 1 {
		return 0, false
	} else if val, err := strconv.ParseFloat(value[0], 64); err == nil {
		return val, true
	} else {
		return 0, false
	}
}

func handleGetMetrics(response *goyave.Response, request *goyave.Request) {
	logDebugf("-------------------\n")
	logDebugf("method : %v\n", request.Method())
	logDebugf("path   : %v\n", request.URL().Path)
	logDebugf("remote : %v\n", request.RemoteAddress())
	scraper := getScraperFromIP(getIPAdress(request.Request()))
	logInfof("scraper : %v\n", scraper)
	for k, v := range request.Header() {
		logDebugf("header  : %s = %s\n", k, v)
	}
	sb := strings.Builder{}
	now := time.Now().UnixMilli()
	// fmt.Fprintf(&sb, `# scraper ID: %s\n`, scraper)
	// events_to_metrics{method="post",code="200"} $value $timestamp
	fmt.Fprintf(&sb, "# HELP events_to_metrics_event_count Scrapers currently being tracked and the events they have yet to collect\n")
	fmt.Fprintf(&sb, "# TYPE events_to_metrics_event_count counter\n")
	if useExplicitTimestamps {
		fmt.Fprintf(&sb, "events_to_metrics_event_count{} %d %d\n", eventCounter.Load(), now)
	} else {
		fmt.Fprintf(&sb, "events_to_metrics_event_count{} %d\n", eventCounter.Load())
	}
	counts := cache.GetScraperEntryCounts()
	if len(counts) > 0 {
		fmt.Fprintf(&sb, "# HELP events_to_metrics_scraper_counts Scrapers currently being tracked and the events they have yet to collect\n")
		fmt.Fprintf(&sb, "# TYPE events_to_metrics_scraper_counts gauge\n")
		for k, v := range counts {
			if useExplicitTimestamps {
				fmt.Fprintf(&sb, "events_to_metrics_scraper_counts{scraper=\"%s\"} %d %d\n", k, v, now)
			} else {
				fmt.Fprintf(&sb, "events_to_metrics_scraper_counts{scraper=\"%s\"} %d\n", k, v)
			}
		}
	}
	metrics := cache.Scrape(scraper)
	if len(metrics) > 0 {
		fmt.Fprintf(&sb, "# HELP events_to_metrics Events registered generically to the conversion service\n")
		fmt.Fprintf(&sb, "# TYPE events_to_metrics gauge\n")
		for k, v := range metrics {
			fmt.Fprintf(&sb, "%s %s\n", k, v)
		}
	}
	response.String(http.StatusOK, sb.String())
	response.Status(http.StatusOK)
}

func getIPAdress(request *http.Request) string {
	for _, header := range []string{"X-Real-Ip", "X-Forwarded-For"} {
		for _, ip := range strings.Split(request.Header.Get(header), ",") {
			ip := net.ParseIP(strings.ReplaceAll(ip, " ", ""))
			if ip != nil {
				return ip.String()
			}
		}
	}
	return request.RemoteAddr
}

func getScraperFromIP(str string) string {
	if strings.Count(str, ".") == 3 { // IPv4
		s := strings.Split(str, ":")
		return s[0]
	} else { //IPv6
		s := strings.Split(str, "]:")
		return strings.ReplaceAll(s[0], "[", "")
	}
}

func logFatalf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func logErrorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

func logWarnf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...))
}

func logInfof(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

func logDebugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}
