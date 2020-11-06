package proxy

import (
	"sync/atomic"
)

var g_sess int64

type ProxyCtx struct {
	Session       int64
	TransferBytes int64
	Request       *ProxyRequest
	Response      *ProxyResponse
}

type ProxyRequest struct {
	Uri     string
	Headers map[string][]string
	Tls     bool
}

type ProxyResponse struct {
	Headers    map[string][]string
	StatusCode int
}

func NewProxyCtx() *ProxyCtx {
	return &ProxyCtx{
		Session: atomic.AddInt64(&g_sess, 1),
	}
}
