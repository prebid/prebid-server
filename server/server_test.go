package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	metricsconfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerShutdown(t *testing.T) {
	server := &http.Server{}
	ln := &mockListener{}

	stopper := make(chan os.Signal)
	done := make(chan struct{})
	go shutdownAfterSignals(server, stopper, done)
	go server.Serve(ln) //nolint: errcheck

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

func TestNewSocketServer(t *testing.T) {
	const mockSocket = "socket_addr:socket_port"
	cfg := new(config.Configuration)
	cfg.UnixSocketName = mockSocket

	mockServer := &http.Server{
		Addr:         cfg.UnixSocketName,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	ret := newSocketServer(cfg, nil)
	require.NotNil(t, ret, "ret : isNil()")

	assert.Equal(t, mockServer.Addr, ret.Addr, fmt.Sprintf("Addr invalide: %v != %v",
		ret.Addr, mockServer.Addr))
	assert.Equal(t, mockServer.ReadTimeout, ret.ReadTimeout, fmt.Sprintf("ReadTimeout invalide: %v != %v",
		ret.ReadTimeout, mockServer.ReadTimeout))
	assert.Equal(t, mockServer.WriteTimeout, ret.WriteTimeout, fmt.Sprintf("WriteTimeout invalide: %v != %v",
		ret.WriteTimeout, mockServer.WriteTimeout))

	ret.Close()
}

func TestNewMainServer(t *testing.T) {
	const (
		mockPort    = 8000         // chose your socket_port
		mockAddress = "prebid.com" // chose your socket_address
	)
	cfg := new(config.Configuration)
	cfg.Port = mockPort
	cfg.Host = mockAddress

	mockServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", mockAddress, mockPort),
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	ret := newMainServer(cfg, nil)
	require.NotNil(t, ret, "ret : isNil()")

	assert.Equal(t, ret.Addr, mockServer.Addr, fmt.Sprintf("Addr invalide: %v != %v",
		ret.Addr, mockServer.Addr))
	assert.Equal(t, ret.ReadTimeout, mockServer.ReadTimeout,
		fmt.Sprintf("ReadTimeout invalide: %v != %v", ret.ReadTimeout, mockServer.ReadTimeout))
	assert.Equal(t, ret.WriteTimeout, mockServer.WriteTimeout,
		fmt.Sprintf("WriteTimeout invalide: %v != %v", ret.WriteTimeout, mockServer.WriteTimeout))

	ret.Close()
}

func TestNewTCPListener(t *testing.T) {
	const mockAddress = ":8000" //:chose your socket_port

	ret, err := newTCPListener(mockAddress, nil)
	assert.Equal(t, nil, err, fmt.Sprintf("err_ : %v", err))
	assert.NotEqual(t, nil, ret, "ret : isNil()")

	if ret != nil {
		ret.Close()
	}
}

func TestNewUnixListener(t *testing.T) {
	const mockFile = "file_referer" // chose your file_referer

	ret, err := newUnixListener(mockFile, nil)
	assert.Equal(t, nil, err, "err_ : NOT-Nil()")
	assert.NotEqual(t, nil, ret, "ret : isNil()")

	if ret != nil {
		ret.Close()
	}
}

func TestNewAdminServer(t *testing.T) {
	const (
		mockHost  = "prebid.com" // chose your host
		mockAdmin = 6060         // chose your admin_port
	)
	cfg := new(config.Configuration)
	cfg.Host = mockHost
	cfg.AdminPort = mockAdmin

	mockServer := &http.Server{
		Addr:    cfg.Host + ":" + strconv.Itoa(cfg.AdminPort),
		Handler: nil,
	}

	ret := newAdminServer(cfg, nil)
	require.NotNil(t, ret, "ret : isNil()")
	assert.Equal(t, mockServer.Addr, ret.Addr, fmt.Sprintf("Addr invalide: %v != %v",
		ret.Addr, mockServer.Addr))

	ret.Close()
}

func TestRunServer(t *testing.T) {
	const mockName = "mockServer_name"

	err := runServer(nil, mockName, nil)
	assert.NotEqual(t, nil, err, "runServer(nil, 'mockName', nil) : didn't trigger any error.")

	s := http.Server{}
	err = runServer(&s, mockName, nil)
	assert.NotEqual(t, nil, err, "runServer(not_nil, 'mockName', nil) : didn't trigger any error.")

	var l net.Listener
	l, _ = net.Listen("error", ":8000")
	err = runServer(&s, mockName, l)
	assert.NotEqual(t, nil, err, "Listen('error', ':8000') : didn't trigger any error.")
}

func TestListen(t *testing.T) {
	var (
		handler, adminHandler http.Handler

		metrics = new(metricsconfig.DetailedMetricsEngine)
		cfg     = &config.Configuration{
			Host:             "prebid.com",
			AdminPort:        6060,
			Port:             8000,
			UnixSocketEnable: false,
			UnixSocketName:   "prebid_socket",
		}
	)

	err := Listen(cfg, handler, adminHandler, metrics)
	assert.NotEqual(t, nil, err, "err : isNil()")
}
