package server

import (
	"net"
	"time"
)

type monitorableConnection struct {
	net.Conn
	onClose func()
}

type monitorableListener struct {
	*net.TCPListener
	onNewConnection func()
}

func (l *monitorableConnection) Close() error {
	l.onClose()
	return l.Conn.Close()
}

func (ln *monitorableListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	ln.onNewConnection()
	return tc, nil
}
