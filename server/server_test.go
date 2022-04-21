package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	metricsconfig "github.com/prebid/prebid-server/metrics/config"
)

func TestNewAdminServer(t *testing.T) {
	cfg := &config.Configuration{
		Host:      "prebid.com",
		AdminPort: 6060,
		Port:      8000,
	}
	server := newAdminServer(cfg, http.HandlerFunc(handler))
	if server.Addr != "prebid.com:6060" {
		t.Errorf("Admin server address should be %s. Got %s", "prebid.com:6060\n", server.Addr)
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
		t.Errorf("Admin server address should be %s. Got %s", "prebid.com:8000\n", server.Addr)
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
		t.Errorf("Unexpected signal: %s\n", sig.String())
	}
	outbound <- s
}

func TestNewSocketServer(t *testing.T) {
	const (
		func_name   = "TestNewSocketServer"
		mock_socket = "socket_addr:socket_port"
	)
	cfg := new(config.Configuration)
	cfg.Socket = mock_socket

	mock_server := &http.Server{
		Addr:         cfg.Socket,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if ret := newSocketServer(cfg, nil); ret != nil {
		switch {
		case ret.Addr != mock_server.Addr:
			t.Errorf("[%s] Addr invalide: %v != %v\n",
				func_name, ret.Addr, mock_server.Addr)
			fallthrough
		case ret.ReadTimeout != mock_server.ReadTimeout:
			t.Errorf("[%s] ReadTimeout invalide: %v != %v\n",
				func_name, ret.ReadTimeout, mock_server.ReadTimeout)
			fallthrough
		case ret.WriteTimeout != mock_server.WriteTimeout:
			t.Errorf("[%s] WriteTimeout invalide: %v != %v\n",
				func_name, ret.WriteTimeout, mock_server.WriteTimeout)
		}
		ret.Close()
	} else {
		t.Errorf("[%s] ret is Nil", func_name)
	}
}

func TestNewMainServer_(t *testing.T) {
	const (
		func_name    = "TestNewMainServer_"
		mock_port    = 8000         // chose your socket_port
		mock_address = "prebid.com" // chose your socket_address
	)
	cfg := new(config.Configuration)
	cfg.Port = mock_port
	cfg.Host = mock_address

	mock_server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", mock_address, mock_port),
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if ret := newMainServer(cfg, nil); ret != nil {
		switch {
		case ret.Addr != mock_server.Addr:
			t.Errorf("[%s] Addr invalide: %v != %v\n",
				func_name, ret.Addr, mock_server.Addr)
		case ret.ReadTimeout != mock_server.ReadTimeout:
			t.Errorf("[%s] ReadTimeout invalide: %v != %v\n",
				func_name, ret.ReadTimeout, mock_server.ReadTimeout)
		case ret.WriteTimeout != mock_server.WriteTimeout:
			t.Errorf("[%s] WriteTimeout invalide: %v != %v\n",
				func_name, ret.WriteTimeout, mock_server.WriteTimeout)
		}
		ret.Close()
	} else {
		t.Errorf("[%s] ret is Nil", func_name)
	}
}

func TestNewTCPListener(t *testing.T) {
	const (
		func_name    = "TestNewTCPListener"
		mock_address = ":8000" //:chose your socket_port
	)

	if ret, err := newTCPListener(mock_address, nil); err != nil {
		t.Errorf("[%s] err_ : %s", func_name, err)
	} else {
		ret.Close()
	}
}

func TestNewUnixListener(t *testing.T) {
	const (
		func_name = "TestNewUnixListener"
		mock_file = "file_referer" // chose your file_referer
	)

	if ret, err := newUnixListener(mock_file, nil); err != nil {
		t.Errorf("[%s] err_ : %s", func_name, err.Error())
	} else {
		ret.Close()
	}
}

func TestNewAdminServer_(t *testing.T) {
	const (
		func_name  = "TestNewAdminServer_"
		mock_host  = "prebid.com" // chose your host
		mock_admin = 6060         // chose your admin_port
	)
	cfg := new(config.Configuration)
	cfg.Host = mock_host
	cfg.AdminPort = mock_admin

	mock_server := &http.Server{
		Addr:    cfg.Host + ":" + strconv.Itoa(cfg.AdminPort),
		Handler: nil,
	}

	if ret := newAdminServer(cfg, nil); ret == nil {
		t.Errorf("[%s] ret : isNil()", func_name)
	} else {
		if ret.Addr != mock_server.Addr {
			t.Errorf("[%s] Addr invalide: %v != %v\n",
				func_name, ret.Addr, mock_server.Addr)
		}
		ret.Close()
	}
}

func TestRunServer(t *testing.T) {
	const (
		func_name = "TestRunServer"
		mock_name = "mock_server_name"
	)

	if err := runServer(nil, mock_name, nil); err == nil {
		t.Errorf("[%s] runServer(nil, 'mock_name', nil) : didn't trigger any error.", func_name)
	}

	s := http.Server{}
	if err := runServer(&s, mock_name, nil); err == nil {
		t.Errorf("[%s] runServer(not_nil, 'mock_name', nil) : didn't trigger any error.", func_name)
	}

	var l net.Listener
	if err := runServer(&s, mock_name, l); err != nil {
		t.Errorf("[%s] runServer(not_nil, 'mock_name', not_nil) : trigger an error.", func_name)
	}
}

func TestListen(t *testing.T) {
	const name = "TestListen"
	var (
		handler       http.Handler
		admin_handler http.Handler

		metrics = new(metricsconfig.DetailedMetricsEngine)
		cfg     = &config.Configuration{
			Host:         "prebid.com",
			AdminPort:    6060,
			Port:         8000,
			EnableSocket: false,
			Socket:       "prebid_socket",
		}
	)

	if e := Listen(cfg, handler, admin_handler, metrics); e != nil {
		t.Errorf("[%s] err_ : %s", name, e.Error())
	}
}
