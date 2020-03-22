package internal

import (
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

func (proxy *ProxyHttpServer) handleHttps(w http.ResponseWriter, r *http.Request, proxyctx *ProxyCtx) {
	zap.S().Infof("[Session: %v] Got request: %v, %v, %v, %v", proxyctx.Sess, r.Method, r.Host, r.URL.Path, r.URL.String())
	start := time.Now()

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
			var wg sync.WaitGroup
			wg.Add(2)
			go copyAndHalfClose(proxyctx, targetServerTCPConn, proxyClientTCPConn, &wg)
			go copyAndHalfClose(proxyctx, proxyClientTCPConn, targetServerTCPConn, &wg)
			wg.Wait()
			proxy.sendToPersistChannel(start, r)
		} else {
			go func() {
				var wg sync.WaitGroup
				wg.Add(2)
				go copyWithoutClose(proxyctx, targetServerTCPConn, proxyClientTCPConn, &wg)
				go copyWithoutClose(proxyctx, proxyClientTCPConn, targetServerTCPConn, &wg)
				wg.Wait()

				targetServerTCPConn.Close()
				proxyClientTCPConn.Close()
				proxy.sendToPersistChannel(start, r)
			}()
		}
	}
}

func (proxy *ProxyHttpServer) sendToPersistChannel(start time.Time, r *http.Request) {
	if proxy.Mc != nil {
		end := time.Now()
		elapsed := end.Sub(start)
		proxy.Mc.httptHistory <- &httpMessage{
			Host:     r.Host,
			Port:     r.URL.Port(),
			Method:   r.Method,
			Path:     r.URL.Path,
			Size:     0,
			Duration: elapsed.Milliseconds(),
			Time:     start.Unix(),
			Scheme:   "https",
		}
	}
}
