package goproxy

import (
	"net/http"
	"github.com/google/logger"
	)

type WrappedHandler struct {
	logg *logger.Logger
}

func (handler *WrappedHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authentication")
	req.Header.Del("Proxy-Authorization")


}

func (handler *WrappedHandler) dumpHTTP(res http.ResponseWriter) {

}