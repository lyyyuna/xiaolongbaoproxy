package proxy

import (
	"crypto/tls"
	"io"
	"net"
)

type HttpsListener struct {
	conn *tls.Conn
}

func (l *HttpsListener) Accept() (net.Conn, error) {
	if l.conn != nil {
		conn := l.conn
		l.conn = nil
		return conn, nil
	} else {
		return nil, io.EOF
	}
}

func (l *HttpsListener) Close() error {
	return nil
}

func (l *HttpsListener) Addr() net.Addr {
	return nil
}
