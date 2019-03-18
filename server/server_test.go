package server

import (
	"net/http"
	"os"
	"testing"

	"github.com/prebid/prebid-server/config"
)

func TestNewAdminServer(t *testing.T) {
	cfg := &config.Configuration{
		Host:      "prebid.com",
		AdminPort: 6060,
		Port:      8000,
	}
	server := newAdminServer(cfg, http.HandlerFunc(handler))
	if server.Addr != "prebid.com:6060" {
		t.Errorf("Admin server address should be %s. Got %s", "prebid.com:6060", server.Addr)
	}
}

func TestNewMainServer(t *testing.T) {
	cfg := &config.Configuration{
		Host:      "prebid.com",
		AdminPort: 6060,
		Port:      8000,
	}
	server := newMainServer(cfg, http.HandlerFunc(handler))
	if server.Addr != "prebid.com:8000" {
		t.Errorf("Admin server address should be %s. Got %s", "prebid.com:8000", server.Addr)
	}
}

func TestServerShutdown(t *testing.T) {
	server := &http.Server{}
	ln := &mockListener{}

	stopper := make(chan os.Signal)
	done := make(chan struct{})
	go shutdownAfterSignals(server, stopper, done)
	go server.Serve(ln)

	stopper <- os.Interrupt
	<-done

	// If the test didn't hang, then we know server.Shutdown really _did_ return, and shutdownAfterSignals
	// passed the message along as expected.
}

func TestWait(t *testing.T) {
	inbound := make(chan os.Signal)
	chan1 := make(chan os.Signal)
	chan2 := make(chan os.Signal)
	chan3 := make(chan os.Signal)
	done := make(chan struct{})

	go forwardSignal(t, done, chan1)
	go forwardSignal(t, done, chan2)
	go forwardSignal(t, done, chan3)

	go func(chan os.Signal) {
		inbound <- os.Interrupt
	}(inbound)

	wait(inbound, done, chan1, chan2, chan3)
	// If this doesn't hang, then wait() is sending and receiving messages as expected.
}

func handler(w http.ResponseWriter, req *http.Request) {

}

// forwardSignal is basically a working mock for shutdownAfterSignals().
// It is used to test wait() effectively
func forwardSignal(t *testing.T, outbound chan<- struct{}, inbound <-chan os.Signal) {
	var s struct{}
	sig := <-inbound
	if sig != os.Interrupt {
		t.Errorf("Unexpected signal: %s", sig.String())
	}
	outbound <- s
}
