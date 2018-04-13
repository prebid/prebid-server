package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
)

// Listen blocks forever, serving PBS requests on the given port. This will block forever, until the process is shut down.
func Listen(cfg *config.Configuration, handler http.Handler) {

	// Run the servers. Fan any process-stopper signals out to each server for graceful shutdowns.
	stopAdmin := make(chan os.Signal)
	stopMain := make(chan os.Signal)
	done := make(chan struct{})

	go runAdmin(cfg, stopAdmin, done)
	go runMain(cfg, handler, stopMain, done)

	wait(done, stopMain, stopAdmin)

	return
}

// runAdmin runs the admin server. This should block forever.
// If an error occurs and it needs to return, a SIGTERM will be sent to stopSignals.
func runAdmin(cfg *config.Configuration, stopper <-chan os.Signal, done chan<- struct{}) {
	uri := cfg.Host + ":" + strconv.Itoa(cfg.AdminPort)
	server := &http.Server{Addr: uri}
	glog.Infof("Admin server starting on: %s", uri)
	go shutdownAfterSignals(server, stopper, done)
	err := server.ListenAndServe()
	glog.Errorf("Admin server quit with error: %v", err)
	return
}

// runMain runs the "main" server. This should block forever, unless something goes wrong.
func runMain(cfg *config.Configuration, handler http.Handler, stopper <-chan os.Signal, done chan<- struct{}) {
	server := &http.Server{
		Addr:         cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		glog.Errorf("Error listening for TCP connections on %s: %v", server.Addr, err)
		return
	}

	// This cast is in Go's core libs as Server.ListenAndServe(), so it _should_ be safe, but just in case...
	if casted, ok := listener.(*net.TCPListener); ok {
		listener = &monitorableListener{casted, func() {}}
	} else {
		glog.Errorf("Golang's core lib didn't return a TCPListener as expected. Connection metrics will not be sent.")
	}

	glog.Infof("Main server starting on: %s", server.Addr)
	go shutdownAfterSignals(server, stopper, done)
	err = server.Serve(listener)
	glog.Errorf("Main server quit with error: %v", err)
	return
}

func wait(done <-chan struct{}, outbound ...chan<- os.Signal) {
	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)
	sig := <-stopSignals

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
