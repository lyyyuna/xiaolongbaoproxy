package proxy

import (
	"github.com/golang/glog"
	"net/http"
	"time"
)

func StartProxy(port string) *http.Server{
	handler := &WrappedHandler{}
	server := &http.Server{
		Addr: ":" + port,
		Handler: handler,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	glog.Info("Proxy server starting..")
	return server
}

