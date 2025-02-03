package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	gometrics "github.com/rcrowley/go-metrics"
)

func TestNormalConnectionMetrics(t *testing.T) {
	doTest(t, true, true)
}

func TestAcceptErrorMetrics(t *testing.T) {
	doTest(t, false, false)
}

func TestCloseErrorMetrics(t *testing.T) {
	doTest(t, true, false)
}

func doTest(t *testing.T, allowAccept bool, allowClose bool) {
	reg := gometrics.NewRegistry()
	me := metrics.NewMetrics(reg, nil, config.DisabledMetrics{}, nil, nil)

	var listener net.Listener = &mockListener{
		listenSuccess: allowAccept,
		closeSuccess:  allowClose,
	}

	listener = &monitorableListener{listener, me}
	conn, err := listener.Accept()
	if !allowAccept {
		if err == nil {
			t.Error("The listener.Accept() error should propagate from the underlying listener.")
		}
		assertCount(t, "When Accept() fails, connection count", me.ConnectionCounter.Count(), 0)
		assertCount(t, "When Accept() fails, Accept() errors", me.ConnectionAcceptErrorMeter.Count(), 1)
		assertCount(t, "When Accept() fails, Close() errors", me.ConnectionCloseErrorMeter.Count(), 0)
		return
	}
	assertCount(t, "When Accept() succeeds, active connections", me.ConnectionCounter.Count(), 1)
	assertCount(t, "When Accept() succeeds, Accept() errors", me.ConnectionAcceptErrorMeter.Count(), 0)

	err = conn.Close()
	if allowClose {
		assertCount(t, "When Accept() and Close() succeed, connection count", me.ConnectionCounter.Count(), 0)
		assertCount(t, "When Accept() and Close() succeed, Accept() errors", me.ConnectionAcceptErrorMeter.Count(), 0)
		assertCount(t, "When Accept() and Close() succeed, Close() errors", me.ConnectionCloseErrorMeter.Count(), 0)
	} else {
		if err == nil {
			t.Error("The connection.Close() error should propagate from the underlying listener.")
		}
		assertCount(t, "When Accept() succeeds sand Close() fails, connection count", me.ConnectionCounter.Count(), 1)
		assertCount(t, "When Accept() succeeds sand Close() fails, Accept() errors", me.ConnectionAcceptErrorMeter.Count(), 0)
		assertCount(t, "When Accept() succeeds sand Close() fails, Close() errors", me.ConnectionCloseErrorMeter.Count(), 1)
	}
}

func assertCount(t *testing.T, context string, actual int64, expected int) {
	t.Helper()
	if actual != int64(expected) {
		t.Errorf("%s: expected %d, got %d", context, expected, actual)
	}
}

type mockListener struct {
	listenSuccess bool
	closeSuccess  bool
}

func (l *mockListener) Accept() (net.Conn, error) {
	if l.listenSuccess {
		return &mockConnection{l.closeSuccess}, nil
	} else {
		return nil, errors.New("Failed to open connection")
	}
}

func (l *mockListener) Close() error {
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return &mockAddr{}
}

type mockConnection struct {
	closeSuccess bool
}

func (c *mockConnection) Read(b []byte) (n int, err error) {
	return len(b), nil
}

func (c *mockConnection) Write(b []byte) (n int, err error) {
	return
}

func (c *mockConnection) Close() error {
	if c.closeSuccess {
		return nil
	} else {
		return errors.New("Failed to close connection.")
	}
}

func (c *mockConnection) LocalAddr() net.Addr {
	return &mockAddr{}
}

func (c *mockConnection) RemoteAddr() net.Addr {
	return &mockAddr{}
}

func (c *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

type mockAddr struct{}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return "192.0.2.1:25"
}
