package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/imuni4fun/fadingMetricsCache"
	"goyave.dev/goyave/v5"
	"goyave.dev/goyave/v5/config"
	"goyave.dev/goyave/v5/util/errors"
)

var cache = fadingMetricsCache.FadingMetricsCache{}

func main() {
	runServer(context.Background())
}

func runServer(ctx context.Context) {
	cache.Configure(ctx, 10*time.Minute, 5, 10000)

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
	for k, v := range request.Request().Form {
		labels[k] = v[0]
		logDebugf("label : %v: %v\n", k, v[0])
	}
	for k, v := range request.Header() {
		logDebugf("header : %s = %s\n", k, v)
	}
	cache.RegisterValue("events_to_metrics", labels, 1)
	response.Status(http.StatusOK)
}

func handleGetMetrics(response *goyave.Response, request *goyave.Request) {
	logDebugf("-------------------\n")
	logDebugf("method : %v\n", request.Method())
	logDebugf("path   : %v\n", request.URL().Path)
	logDebugf("remote : %v\n", request.RemoteAddress())
	for k, v := range request.Header() {
		logDebugf("header : %s = %s\n", k, v)
	}
	logInfof("scraper: %v\n", getIPAdress(request.Request()))
	leadIn := `// # HELP events_to_metrics Events registered generically to the conversion service
// # scraper ID: %s
// # TYPE events_to_metrics guage
// events_to_metrics{method="post",code="200"} $value $timestamp`
	scraper := request.RemoteAddress()
	metrics := cache.Scrape(scraper)
	sb := strings.Builder{}
	fmt.Fprintf(&sb, leadIn, scraper)
	for k, v := range metrics {
		fmt.Fprintf(&sb, "\n%s %s", k, v)
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
