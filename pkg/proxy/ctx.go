package proxy

import (
	"sync/atomic"
)

var g_sess int64

type ProxyCtx struct {
	session       int64
	TransferBytes int64
}

func NewProxyCtx() *ProxyCtx {
	return &ProxyCtx{
		session: atomic.AddInt64(&g_sess, 1),
	}
}
