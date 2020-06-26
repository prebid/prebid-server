package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics"
	metricsconfig "github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics/config"
	"github.com/golang/glog"
)

// Listen blocks forever, serving PBS requests on the given port. This will block forever, until the process is shut down.
func Listen(cfg *config.Configuration, handler http.Handler, adminHandler http.Handler, metrics *metricsconfig.DetailedMetricsEngine) {
	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	// Run the servers. Fan any process-stopper signals out to each server for graceful shutdowns.
	stopAdmin := make(chan os.Signal)
	stopMain := make(chan os.Signal)
	stopPrometheus := make(chan os.Signal)
	done := make(chan struct{})

	adminServer := newAdminServer(cfg, adminHandler)
	go shutdownAfterSignals(adminServer, stopAdmin, done)

	mainServer := newMainServer(cfg, handler)
	go shutdownAfterSignals(mainServer, stopMain, done)

	mainListener, err := newListener(mainServer.Addr, metrics)
	if err != nil {
		glog.Errorf("Error listening for TCP connections on %s: %v for main server", mainServer.Addr, err)
		return
	}
	adminListener, err := newListener(adminServer.Addr, nil)
	if err != nil {
		glog.Errorf("Error listening for TCP connections on %s: %v for admin server", adminServer.Addr, err)
		return
	}
	go runServer(mainServer, "Main", mainListener)
	go runServer(adminServer, "Admin", adminListener)

	if cfg.Metrics.Prometheus.Port != 0 {
		prometheusServer := newPrometheusServer(cfg, metrics)
		go shutdownAfterSignals(prometheusServer, stopPrometheus, done)
		prometheusListener, err := newListener(prometheusServer.Addr, nil)
		if err != nil {
			glog.Errorf("Error listening for TCP connections on %s: %v for prometheus server", adminServer.Addr, err)
			return
		}
		go runServer(prometheusServer, "Prometheus", prometheusListener)

		wait(stopSignals, done, stopMain, stopAdmin, stopPrometheus)
	} else {
		wait(stopSignals, done, stopMain, stopAdmin)
	}
	return
}

func newAdminServer(cfg *config.Configuration, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    cfg.Host + ":" + strconv.Itoa(cfg.AdminPort),
		Handler: handler,
	}
}

func newMainServer(cfg *config.Configuration, handler http.Handler) *http.Server {
	var serverHandler = handler
	if cfg.EnableGzip {
		serverHandler = gziphandler.GzipHandler(handler)
	}

	return &http.Server{
		Addr:         cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Handler:      serverHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

}

func runServer(server *http.Server, name string, listener net.Listener) {
	glog.Infof("%s server starting on: %s", name, server.Addr)
	err := server.Serve(listener)
	glog.Errorf("%s server quit with error: %v", name, err)
}

func newListener(address string, metrics pbsmetrics.MetricsEngine) (net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("Error listening for TCP connections on %s: %v", address, err)
	}

	// This cast is in Go's core libs as Server.ListenAndServe(), so it _should_ be safe, but just in case it changes in a future version...
	if casted, ok := ln.(*net.TCPListener); ok {
		ln = &tcpKeepAliveListener{casted}
	} else {
		glog.Warning("net.Listen(\"tcp\", \"addr\") didn't return a TCPListener as it did in Go 1.9. Things will probably work fine... but this should be investigated.")
	}

	if metrics != nil {
		ln = &monitorableListener{ln, metrics}
	}

	return ln, nil
}

func wait(inbound <-chan os.Signal, done <-chan struct{}, outbound ...chan<- os.Signal) {
	sig := <-inbound

	for i := 0; i < len(outbound); i++ {
		go sendSignal(outbound[i], sig)
	}

	for i := 0; i < len(outbound); i++ {
		<-done
	}
}

func shutdownAfterSignals(server *http.Server, stopper <-chan os.Signal, done chan<- struct{}) {
	sig := <-stopper

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var s struct{}
	glog.Infof("Stopping %s because of signal: %s", server.Addr, sig.String())
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("Failed to shutdown %s: %v", server.Addr, err)
	}
	done <- s
}

func sendSignal(to chan<- os.Signal, sig os.Signal) {
	to <- sig
}
