package internal

import (
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
)

// After a call to Hijack the HTTP server library will not do anything else with the connection.
// It becomes the caller's responsibility to manage and close the connection.
// So in this function, we have to write 5xx back, and close the connection
func httpError(w io.WriteCloser, ctx *ProxyCtx, err error) {
	if _, err := io.WriteString(w, "HTTP/1.1 502 Bad Gateway\r\n\r\n"); err != nil {
		zap.S().Errorf("[Session: %v] Fail to respond to client, the error is: %v", ctx.Sess, err)
	}
	if err := w.Close(); err != nil {
		zap.S().Errorf("[Session: %v] Fail to close the client connection, the error is: %v", ctx.Sess, err)
	}
}

// half close, as both sides are a normal connection
func copyAndHalfClose(proxyctx *ProxyCtx, dst, src *net.TCPConn) {
	if _, err := io.Copy(dst, src); err != nil {
		zap.S().Errorf("[Session: %v] Error copying to clients, the error is: %v", proxyctx.Sess, err)
	}

	dst.CloseWrite()
	src.CloseRead()
}

// copy without close, one side is abnormal,
// so let the caller close connection
func copyWithoutClose(proxyctx *ProxyCtx, dst, src *net.TCPConn, wg *sync.WaitGroup) {
	if _, err := io.Copy(dst, src); err != nil {
		zap.S().Errorf("[Session: %v] Error copying to clients, the error is: %v", proxyctx.Sess, err)
	}

	wg.Done()
}
