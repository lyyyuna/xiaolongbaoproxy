package proxy

import (
	"crypto/tls"
	"net"
)

type HttpsListener struct {
	conn *tls.Conn
}

func (l *HttpsListener) Accept() (net.Conn, error) {
	return l.conn, nil
}

func (l *HttpsListener) Close() error {
	return l.Close()
}

func (l *HttpsListener) Addr() net.Addr {
	return nil
}
