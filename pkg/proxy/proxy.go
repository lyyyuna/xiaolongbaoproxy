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
	"xiaolongbaoproxy/pkg/keycache"

	"go.uber.org/zap"
)

type ProxyServer struct {
	Mitm           bool
	Tr             *http.Transport
	Hook           func(*ProxyCtx)
	Cert           *key.Certificate
	PrivateKey     *key.PrivateKey
	TlsConfig      *tls.Config
	certCache      *keycache.CertCache
	fakeServerPool *sync.Pool
}

var hasPort = regexp.MustCompile(`:\d+$`)

func NewProxyServer(hook func(*ProxyCtx)) *ProxyServer {
	return &ProxyServer{
		Mitm: false,
		Tr:   &http.Transport{},
		Hook: hook,
	}
}

func NewMitmProxyServer(certpath, pkpath string, cachepath string, hook func(*ProxyCtx)) *ProxyServer {
	cert, err := key.LoadCertificateFromFile(certpath)
	if err != nil {
		zap.S().Fatalf("read cert failed: %v", err)
	}
	pk, err := key.LoadPKFromFile(pkpath)
	if err != nil {
		zap.S().Fatalf("read key failed: %v", err)
	}
	cache, err := keycache.NewCertCache(cachepath, cert, pk)
	if err != nil {
		zap.S().Fatalf("initalize cert cache failed: %v", err)
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
		certCache: cache,
		fakeServerPool: &sync.Pool{
			New: func() interface{} {
				return new(http.Server)
			},
		},
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewProxyCtx()
	zap.S().Infof("[%v] got request: %v, %v, from %v", ctx.Session, r.Method, r.URL, r.RemoteAddr)
	if r.Method == "CONNECT" {
		p.TransferHttps(ctx, w, r)
	} else {
		p.TransferPlainText(ctx, w, r)
	}
}

func (p *ProxyServer) TransferPlainText(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p.Hook != nil {
			p.Hook(ctx)
		}
	}()

	ctx.Request.Host = r.Host
	ctx.Request.Url = r.URL.String()
	ctx.Request.Headers = r.Header

	res, err := p.Tr.RoundTrip(r)
	if err != nil {
		zap.S().Errorf("[%v] response from %v error", ctx.Session, r.URL)
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
		zap.S().Errorf("[%v] send response back to client failed: %v", ctx.Session, err)
		http.Error(w, "", res.StatusCode)
		return
	}

	zap.S().Debugf("[%v] transfer %v bytes", ctx.Session, nb)
	ctx.TransferBytes = nb
	ctx.Response.Headers = res.Header
	ctx.Response.StatusCode = res.StatusCode
}

func (p *ProxyServer) TransferHttps(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	// defer func() {
	// 	if p.Hook != nil {
	// 		p.Hook(ctx, r)
	// 	}
	// }()

	ctx.Request.Tls = true

	hj, ok := w.(http.Hijacker)
	if !ok {
		zap.S().Errorf("[%v] the http server does not support hijacker", ctx.Session)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	connFromClient, _, err := hj.Hijack()
	if err != nil {
		zap.S().Errorf("[%v] fail to hijack the connection: %v", ctx.Session, err)
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
			zap.S().Errorf("[%v] fail to connect to remote: %v", ctx.Session, err)
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

		// get from key cache
		keypair, err := p.certCache.GetKeyPair(host)
		if keypair == nil {
			keypair = &tls.Certificate{}
		}
		if err == nil && keypair != nil {
			zap.S().Debugf("[%v][tls] found one key pair in cache for: %v", ctx.Session, host)
		}
		if err != nil {
			zap.S().Infof("[%v][tls] key not found for %v: %v", ctx.Session, host, err)
			signedcert, signedkey, err := key.CertificateForKey(host, p.PrivateKey, p.Cert)
			if err != nil {
				zap.S().Errorf("[%v][tls] fail to generate a key for: %v, reason: %v", ctx.Session, host, err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			p.certCache.SetKeyPair(host, signedcert.DerBytes, signedkey.PEMEncoded())
			*keypair, err = tls.X509KeyPair(signedcert.PEMEncoded(), signedkey.PEMEncoded())

			if err != nil {
				zap.S().Errorf("[%v][tls] fail to generate a keypair for: %v", ctx.Session, host)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}

		newTlsConfig := &tls.Config{
			CipherSuites:             p.TlsConfig.CipherSuites,
			PreferServerCipherSuites: p.TlsConfig.PreferServerCipherSuites,
			InsecureSkipVerify:       p.TlsConfig.InsecureSkipVerify,
			Certificates:             []tls.Certificate{*keypair},
		}
		tlsConnFromClient := tls.Server(connFromClient, newTlsConfig)
		httpsListener := &HttpsListener{conn: tlsConnFromClient}
		httpsHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			p.TransferPlainTextToHttpsRemote(ctx, rw, r)
		})

		connFromClient.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		singleServ := p.newSingleUseTlsServer()
		defer p.fakeServerPool.Put(singleServ)
		singleServ.Handler = httpsHandler
		singleServ.Serve(httpsListener)
	}
}

func (p *ProxyServer) TransferPlainTextToHttpsRemote(ctx *ProxyCtx, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p.Hook != nil {
			p.Hook(ctx)
		}
	}()

	ctx.Request.Host = r.Host
	ctx.Request.Url = r.URL.String()
	ctx.Request.Headers = r.Header

	host := r.Host
	if !hasPort.MatchString(host) {
		host += ":443"
	}
	connRemote, err := tls.Dial("tcp", host, p.TlsConfig)
	if err != nil {
		zap.S().Errorf("[%v][tls] fail to dial to : %v, reason: %v", ctx.Session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer connRemote.Close()

	// remove some headers
	p.removeHeaders(r)
	if err = r.Write(connRemote); err != nil {
		zap.S().Errorf("[%v][tls] fail to send request to : %v, reason: %v", ctx.Session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	respRemote, err := http.ReadResponse(bufio.NewReader(connRemote), r)
	if err != nil && err != io.EOF {
		zap.S().Errorf("[%v][tls] fail to read response from : %v, reason: %v", ctx.Session, host, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	for k, vs := range respRemote.Header {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	// Force connection close otherwise chrome will keep CONNECT tunnel open forever
	respRemote.Header.Set("Connection", "close")
	w.WriteHeader(respRemote.StatusCode)
	nb, err := io.Copy(w, respRemote.Body)
	if err != nil {
		zap.S().Errorf("[%v][tls] send response back to client failed: %v", ctx.Session, err)
		http.Error(w, "", respRemote.StatusCode)
		return
	}
	// defer respRemote.Body.Close() should NOT close, or tls connection will break
	zap.S().Debugf("[%v][tls] transfer %v bytes", ctx.Session, nb)
	ctx.TransferBytes = nb
	ctx.Response.Headers = respRemote.Header
	ctx.Response.StatusCode = respRemote.StatusCode
}

func copyWithWait(ctx *ProxyCtx, dst, src *net.TCPConn, wg *sync.WaitGroup) {
	nb, err := io.Copy(dst, src)
	if err != nil && nb == 0 {
		zap.S().Errorf("[%v] transfer encountering error: %v", ctx.Session, err)
	}
	dst.CloseWrite()
	src.CloseRead()
	wg.Done()
}

func (p *ProxyServer) removeHeaders(r *http.Request) {
	r.RequestURI = ""
	r.Header.Del("Accept-Encoding")
	// curl can add that, see
	// https://jdebp.eu./FGA/web-proxy-connection-header.html
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	// Connection, Authenticate and Authorization are single hop Header:
	// http://www.w3.org/Protocols/rfc2616/rfc2616.txt
	// 14.10 Connection
	//   The Connection general-header field allows the sender to specify
	//   options that are desired for that particular connection and MUST NOT
	//   be communicated by proxies over further connections.

	// When server reads http request it sets req.Close to true if
	// "Connection" header contains "close".
	// https://github.com/golang/go/blob/master/src/net/http/request.go#L1080
	// Later, transfer.go adds "Connection: close" back when req.Close is true
	// https://github.com/golang/go/blob/master/src/net/http/transfer.go#L275
	// That's why tests that checks "Connection: close" removal fail
	if r.Header.Get("Connection") == "close" {
		r.Close = false
	}
	r.Header.Del("Connection")
}

func (p *ProxyServer) newSingleUseTlsServer() *http.Server {
	fake := p.fakeServerPool.Get().(*http.Server)
	fake.ReadTimeout = 10 * time.Second
	fake.ReadHeaderTimeout = 5 * time.Second
	fake.WriteTimeout = 10 * time.Second
	fake.IdleTimeout = 10 * time.Second
	return fake
}
