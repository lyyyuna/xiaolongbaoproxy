package proxy

import (
	"net/http"
)

type ProxyServer struct {
	Mitm bool
}

func NewProxyServer(mitm bool) *ProxyServer {
	return &ProxyServer{
		Mitm: mitm,
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		p.TransferHttps(w, r)
	} else {
		p.TransferPlainText(w, r)
	}
}

func (p *ProxyServer) TransferPlainText(w http.ResponseWriter, r *http.Request) {

}

func (p *ProxyServer) TransferHttps(w http.ResponseWriter, r *http.Request) {

}
