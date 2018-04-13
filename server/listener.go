package server

import (
	"net"
	"time"

	"github.com/prebid/prebid-server/pbsmetrics"
)

type monitorableConnection struct {
	net.Conn
	metrics pbsmetrics.MetricsEngine
}

type monitorableListener struct {
	*net.TCPListener
	metrics pbsmetrics.MetricsEngine
}

func (l *monitorableConnection) Close() error {
	// TODO: Log the connection closed
	return l.Conn.Close()
}

func (ln *monitorableListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	// TODO: Log the connection open
	return &monitorableConnection{
		tc,
		ln.metrics,
	}, nil
}
