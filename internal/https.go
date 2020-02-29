package internal

import (
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"sync"
	"time"
)

func (proxy *ProxyHttpServer) handleHttps(w http.ResponseWriter, r *http.Request, proxyctx *ProxyCtx) {
	log.Infof("[Session: %v] Got request: %v, %v, %v, %v", proxyctx.Sess, r.Method, r.Host, r.URL.Path, r.URL.String())

	hij, ok := w.(http.Hijacker)
	if !ok {
		panic("The http server does not suppor hijacking.")
	}

	// After a call to Hijack the HTTP server library will not do anything else with the connection.
	// It becomes the caller's responsibility to manage and close the connection.
	proxyClientConn, _, err := hij.Hijack()
	if err != nil {
		panic("Fail to hijack the connection, the err is " + err.Error())
	}

	host := r.URL.Host
	switch proxy.ProxyType {
	case ConnectNormal:
		if !hasPort.MatchString(host) {
			if r.URL.Scheme == "http" {
				host += ":80"
			} else {
				host += ":443"
			}
		}

		targetServerConn, err := net.DialTimeout("tcp", host, 5*time.Second)

		if err != nil {
			httpError(proxyClientConn, proxyctx, err)
			return
		}

		// Now,
		// 1. remote target server has accept our request
		// 2. we are able to receive client's following connectoins
		proxyClientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		targetServerTCPConn, targetOk := targetServerConn.(*net.TCPConn)
		proxyClientTCPConn, clientOk := proxyClientConn.(*net.TCPConn)
		if targetOk && clientOk {
			go copyAndHalfClose(proxyctx, targetServerTCPConn, proxyClientTCPConn)
			go copyAndHalfClose(proxyctx, proxyClientTCPConn, targetServerTCPConn)
		} else {
			go func() {
				var wg sync.WaitGroup
				wg.Add(2)
				go copyWithoutClose(proxyctx, targetServerTCPConn, proxyClientTCPConn, &wg)
				go copyWithoutClose(proxyctx, proxyClientTCPConn, targetServerTCPConn, &wg)
				wg.Wait()

				targetServerTCPConn.Close()
				proxyClientTCPConn.Close()
			}()
		}
	}
}
