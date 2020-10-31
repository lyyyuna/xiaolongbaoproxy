package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"xiaolongbaoproxy/pkg/key"

	"go.uber.org/zap"
)

type ProxyServer struct {
	Mitm       bool
	Tr         *http.Transport
	Hook       func(*ProxyCtx, *http.Request)
	Cert       *key.Certificate
	PrivateKey *key.PrivateKey
	TlsConfig  *tls.Config
}

var hasPort = regexp.MustCompile(`:\d+$`)

func NewProxyServer(hook func(*ProxyCtx, *http.Request)) *ProxyServer {
	return &ProxyServer{
		Mitm: false,
		Tr:   &http.Transport{},
		Hook: hook,
	}
}

func NewMitmProxyServer(certpath, pkpath string, cachepath string, hook func(*ProxyCtx, *http.Request)) *ProxyServer {
	cert, err := key.LoadCertificateFromFile(certpath)
	if err != nil {
		zap.S().Fatal("read cert failed")
	}
	pk, err := key.LoadPKFromFile(pkpath)
	if err != nil {
		zap.S().Fatal("read key failed")
	}
	return &ProxyServer{
		Mitm:       true,
		Tr:         &http.Transport{},
		Hook:       hook,
		Cert:       cert,
		PrivateKey: pk,
		TlsConfig: &tls.Config{
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			PreferServerCipherSuites: true,
			InsecureSkipVerify:       false},
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
	if err != nil {
		zap.S().Errorf("[%v] response from %v error", ctx.session, r.URL)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	// defer res.Body.Close() should not close,
	for k, vs := range res.Header {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)
	nb, err := io.Copy(w, res.Body)
	if err != nil {
		zap.S().Errorf("[%v] send response back to client failed: %v", ctx.session, err)
		http.Error(w, "", res.StatusCode)
		return
	}
	zap.S().Infof("[%v] transfer %v bytes", ctx.session, nb)
	ctx.TransferBytes = nb
}

func (p *ProxyServer) TransferHttps(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	// defer func() {
	// 	if p.Hook != nil {
	// 		p.Hook(ctx, r)
	// 	}
	// }()

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
	// should not close, or the mitm proxy server (a goroutine) will use
	// a close connection
	// defer connFromClient.Close()

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
			// connToRemote.Close()
			return
		}
		defer connToRemote.Close()

		connFromClient.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		var wg sync.WaitGroup
		wg.Add(2)
		connToRemoteTcp, _ := connToRemote.(*net.TCPConn)
		connFromClientTcp, _ := connFromClient.(*net.TCPConn)
		go copyWithWait(ctx, connToRemoteTcp, connFromClientTcp, &wg)
		go copyWithWait(ctx, connFromClientTcp, connToRemoteTcp, &wg)
		wg.Wait()
	} else {
		addr := r.Host
		host := strings.Split(addr, ":")[0]
		signedcert, err := key.CertificateForKey(host, p.PrivateKey, p.Cert)
		if err != nil {
			zap.S().Errorf("[%v] fail to generate a key for: %v, reason: %v", ctx.session, host, err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		keypair, err := tls.X509KeyPair(signedcert.PEMEncoded(), p.PrivateKey.PEMEncoded())

		if err != nil {
			zap.S().Errorf("[%v] fail to generate a keypair for: %v", ctx.session, host)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		newTlsConfig := &tls.Config{
			CipherSuites:             p.TlsConfig.CipherSuites,
			PreferServerCipherSuites: p.TlsConfig.PreferServerCipherSuites,
			InsecureSkipVerify:       p.TlsConfig.InsecureSkipVerify,
			Certificates:             []tls.Certificate{keypair},
		}
		tlsConnFromClient := tls.Server(connFromClient, newTlsConfig)
		httpsListener := &HttpsListener{conn: tlsConnFromClient}
		httpsHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			p.TransferPlainTextToHttpsRemote(ctx, rw, r)
		})

		connFromClient.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		http.Serve(httpsListener, httpsHandler)
	}
}

func (p *ProxyServer) TransferPlainTextToHttpsRemote(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p.Hook != nil {
			p.Hook(ctx, r)
		}
	}()

	host := r.Host
	if !hasPort.MatchString(host) {
		host += ":443"
	}
	connRemote, err := tls.Dial("tcp", host, p.TlsConfig)
	if err != nil {
		zap.S().Errorf("[%v][tls] fail to dial to : %v, reason: %v", ctx.session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer connRemote.Close()

	if err = r.Write(connRemote); err != nil {
		zap.S().Errorf("[%v][tls] fail to send request to : %v, reason: %v", ctx.session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	respRemote, err := http.ReadResponse(bufio.NewReader(connRemote), r)
	if err != nil && err != io.EOF {
		zap.S().Errorf("[%v][tls] fail to read response from : %v, reason: %v", ctx.session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	for k, vs := range respRemote.Header {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(respRemote.StatusCode)
	nb, err := io.Copy(w, respRemote.Body)
	if err != nil {
		zap.S().Errorf("[%v][tls] send response back to client failed: %v", ctx.session, err)
		http.Error(w, "", respRemote.StatusCode)
		return
	}
	// defer respRemote.Body.Close() should NOT close, or tls connection will break
	zap.S().Infof("[%v][tls] transfer %v bytes", ctx.session, nb)
	ctx.TransferBytes = nb
}

func copyWithWait(ctx *ProxyCtx, dst, src *net.TCPConn, wg *sync.WaitGroup) {
	nb, err := io.Copy(dst, src)
	if err != nil && nb == 0 {
		zap.S().Errorf("[%v] transfer encountering error: %v", ctx.session, err)
	}
	dst.CloseWrite()
	src.CloseRead()
	wg.Done()
}
