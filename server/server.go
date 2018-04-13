package server

import (
	"context"
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
func Listen(cfg *config.Configuration, handler http.Handler) error {
	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	// Run the servers. Fan any process-stopper signals out to each server for graceful shutdowns.
	stopAdmin := make(chan struct{})
	stopMain := make(chan struct{})
	go dispatch(stopSignals, stopMain, stopAdmin)
	go runAdmin(cfg, stopAdmin)
	go runMain(cfg, handler, stopMain)

	<-stopSignals
	return nil
}

func dispatch(inbound <-chan os.Signal, outbound ...chan<- struct{}) {
	var s struct{}
	for {
		<-inbound
		done := make(chan bool)
		for i := 0; i < len(outbound); i++ {
			go func() {
				outbound[i] <- s
				done <- true
			}()
		}
		for i := 0; i < len(outbound); i++ {
			<-done
		}
	}
}

// runAdmin runs the admin server. This should block forever.
// If an error occurs and it needs to return, a SIGTERM will be sent to stopSignals.
func runAdmin(cfg *config.Configuration, stopper <-chan struct{}) {
	uri := cfg.Host + ":" + strconv.Itoa(cfg.AdminPort)
	server := &http.Server{Addr: uri}
	glog.Infof("Admin server starting on: %s", uri)
	go watchForStops(server, stopper)
	err := server.ListenAndServe()
	glog.Errorf("Admin server quit with error: %v", err)
	return
}

// runMain runs the "main" server. This should block forever.
// If an error occurs and it returns, a SIGTERM will be sent to stopSignals.
func runMain(cfg *config.Configuration, handler http.Handler, stopper <-chan struct{}) {
	server := &http.Server{
		Addr:         cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	glog.Infof("Main server starting on: %s", server.Addr)
	go watchForStops(server, stopper)
	err := server.ListenAndServe()
	glog.Errorf("Main server quit with error: %v", err)
	return
}

func watchForStops(server *http.Server, stopper <-chan struct{}) {
	<-stopper
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("Failed to stop server %s : %v", server.Addr, err)
	}
}
