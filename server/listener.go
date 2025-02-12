package server

import (
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/metrics"
)

// monitorableListener tracks any opened connections in the metrics.
type monitorableListener struct {
	net.Listener
	metrics metrics.MetricsEngine
}

// monitorableConnection tracks any closed connections in the metrics.
type monitorableConnection struct {
	net.Conn
	metrics metrics.MetricsEngine
}

func (l *monitorableConnection) Close() error {
	err := l.Conn.Close()
	if err == nil {
		l.metrics.RecordConnectionClose(true)
	} else {
		// If the connection was closed by the client, it's not a real/actionable error.
		// Although there are no official APIs to detect this, this ridiculous workaround appears
		// in the core Go libs: https://github.com/golang/go/issues/4373#issuecomment-347680321
		errString := err.Error()
		if !strings.Contains(errString, "use of closed network connection") {
			glog.Errorf("Error closing connection: %s", errString)
		}
		l.metrics.RecordConnectionClose(false)
	}
	return err
}

func (ln *monitorableListener) Accept() (net.Conn, error) {
	tc, err := ln.Listener.Accept()
	if err != nil {
		glog.Errorf("Error accepting connection: %v", err)
		ln.metrics.RecordConnectionAccept(false)
		return tc, err
	}
	ln.metrics.RecordConnectionAccept(true)
	return &monitorableConnection{
		tc,
		ln.metrics,
	}, nil
}

// tcpKeepAliveListener is copy/pasted from the implementation here: https://golang.org/pkg/net/http/#Server.ListenAndServe
// Since it's not public, the best we can do is copy/paste it here.
//
// We should revisit this after Go 1.11. See also:
// - https://github.com/golang/go/issues/23378
// - https://github.com/golang/go/issues/23459
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

type unixListener struct{ *net.UnixListener }

func (ln unixListener) Accept() (net.Conn, error) { return ln.AcceptUnix() }
