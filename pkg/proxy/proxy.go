package proxy

import (
	"io"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ProxyServer struct {
	Mitm bool
	Tr   *http.Transport
	Hook func(*ProxyCtx, *http.Request)
}

var hasPort = regexp.MustCompile(`:\d+$`)

func NewProxyServer(mitm bool, hook func(*ProxyCtx, *http.Request)) *ProxyServer {
	return &ProxyServer{
		Mitm: mitm,
		Tr:   &http.Transport{},
		Hook: hook,
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewProxyCtx()
	zap.S().Debugf("[%v] got request: %v, %v", ctx.session, r.Method, r.URL)
	if r.Method == "CONNECT" {
		p.TransferHttps(ctx, w, r)
	} else {
		p.TransferPlainText(ctx, w, r)
	}
}

func (p *ProxyServer) TransferPlainText(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p.Hook != nil {
			p.Hook(ctx, r)
		}
	}()

	res, err := p.Tr.RoundTrip(r)
	defer res.Body.Close()
	if err != nil {
		zap.S().Errorf("[%v] response from %v error", ctx.session, res.Request.URL)
		http.Error(w, err.Error(), res.StatusCode)
		return
	}
	w.WriteHeader(res.StatusCode)
	nb, err := io.Copy(w, res.Body)
	if err != nil {
		zap.S().Errorf("[%v] send response back to client failed: %v", ctx.session, err)
		http.Error(w, err.Error(), res.StatusCode)
		return
	}
	zap.S().Infof("[%v] transfer %v bytes", ctx.session, nb)
	ctx.TransferBytes = nb
}

func (p *ProxyServer) TransferHttps(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p.Hook != nil {
			p.Hook(ctx, r)
		}
	}()

	hj, ok := w.(http.Hijacker)
	if !ok {
		zap.S().Errorf("[%v] the http server does not support hijacker", ctx.session)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	connFromClient, _, err := hj.Hijack()
	if err != nil {
		zap.S().Errorf("[%v] fail to hijack the connection: %v", ctx.session, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	host := r.URL.Host
	if !p.Mitm {
		if !hasPort.MatchString(host) {
			if r.URL.Scheme == "http" {
				host += ":80"
			} else if r.URL.Scheme == "https" {
				host += ":443"
			}
		}
		connToRemote, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			zap.S().Errorf("[%v] fail to connect to remote: %v", ctx.session, err)
			io.WriteString(w, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
			connToRemote.Close()
			return
		}

		connFromClient.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		var wg sync.WaitGroup
		wg.Add(2)
		go copyWithWait(ctx, connToRemote, connFromClient, &wg)
		go copyWithWait(ctx, connFromClient, connToRemote, &wg)
		wg.Wait()
	}
}

func copyWithWait(ctx *ProxyCtx, dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	_, err := io.Copy(dst, src)
	if err != nil {
		zap.S().Errorf("[%v] transfer encountering error: %v", ctx.session, err)
	}
	wg.Done()
}
