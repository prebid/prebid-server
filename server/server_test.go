package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

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

func Test_newSocketServer(t *testing.T) {
	cfg := new(config.Configuration)
	cfg.Socket = "chose_your_socket_addr:chose_your_socket_port"

	mock_server := &http.Server{
		Addr:         cfg.Socket,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if ret := newSocketServer(cfg, nil); ret != nil {
		switch {
		case ret.Addr != mock_server.Addr:
			t.Errorf("[Test_newSocketServer] Addr invalide: %v != %v\n",
				ret.Addr, mock_server.Addr)
			fallthrough
		case ret.ReadTimeout != mock_server.ReadTimeout:
			t.Errorf("[Test_newSocketServer] ReadTimeout invalide: %v != %v\n",
				ret.ReadTimeout, mock_server.ReadTimeout)
			fallthrough
		case ret.WriteTimeout != mock_server.WriteTimeout:
			t.Errorf("[Test_newSocketServer] WriteTimeout invalide: %v != %v\n",
				ret.WriteTimeout, mock_server.WriteTimeout)
		}
		ret.Close()
	} else {
		t.Errorf("[Test_newSocketServer] ret is Nil")
	}
}

func Test_newMainServer(t *testing.T) {
	const (
		chose_your_socket_port    = 8000                        //chose_your_socket_port
		chose_your_socket_address = "chose_your_socket_address" //chose_your_socket_address
	)
	cfg := new(config.Configuration)
	cfg.Port = chose_your_socket_port
	cfg.Host = chose_your_socket_address

	mock_server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", chose_your_socket_address, chose_your_socket_port),
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if ret := newMainServer(cfg, nil); ret != nil {
		switch {
		case ret.Addr != mock_server.Addr:
			t.Errorf("[Test_newMainServer] Addr invalide: %v != %v\n",
				ret.Addr, mock_server.Addr)
			fallthrough
		case ret.ReadTimeout != mock_server.ReadTimeout:
			t.Errorf("[Test_newMainServer] ReadTimeout invalide: %v != %v\n",
				ret.ReadTimeout, mock_server.ReadTimeout)
			fallthrough
		case ret.WriteTimeout != mock_server.WriteTimeout:
			t.Errorf("[Test_newMainServer] WriteTimeout invalide: %v != %v\n",
				ret.WriteTimeout, mock_server.WriteTimeout)
		}
		ret.Close()
	} else {
		t.Errorf("[Test_newSocketServer] ret is Nil")
	}
}

func Test_newTCPListener(t *testing.T) {
	const mock_address_value = ":8000" //:chose_your_socket_port

	if ret, err := newTCPListener(mock_address_value, nil); err != nil {
		t.Error("[Test_newTCPListener] err_ :", err)
	} else {
		ret.Close()
	}
}

func Test_newUnixListener(t *testing.T) {
	const mock_file_referer = "chose_your_file_referer"

	if ret, err := newUnixListener(mock_file_referer, nil); err != nil {
		t.Error("[Test_newUnixListener] err_ :", err)
	} else {
		ret.Close()
	}
}

func Test_newAdminServer(t *testing.T) {
	const (
		mock_host_value       = "chose_your_host_value"
		mock_admin_port_value = 42 //chose your admin_port_value
	)
	cfg := new(config.Configuration)
	cfg.Host = mock_host_value
	cfg.AdminPort = mock_admin_port_value

	mock_server := &http.Server{
		Addr:    cfg.Host + ":" + strconv.Itoa(cfg.AdminPort),
		Handler: nil,
	}

	if ret := newAdminServer(cfg, nil); ret == nil {
		t.Error("[Test_newAdminServer] ret : isNil()")
	} else {
		if ret.Addr != mock_server.Addr {
			t.Errorf("[Test_newAdminServer] Addr invalide: %v != %v\n",
				ret.Addr, mock_server.Addr)
		}
		ret.Close()
	}
}

func Test_runServer(t *testing.T) {
	const mock_server_name = "mock_server_name"

	if err := runServer(nil, mock_server_name, nil); err == nil {
		t.Error("[Test_runServer] runServer(nil, 'mock_server_name', nil) : didn't trigger any error.")
	}

	s := http.Server{}
	if err := runServer(&s, mock_server_name, nil); err == nil {
		t.Error("[Test_runServer] runServer(not_nil, 'mock_server_name', nil) : didn't trigger any error.")
	}
}
