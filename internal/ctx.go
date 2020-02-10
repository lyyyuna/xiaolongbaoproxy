package internal

import "net/http"

type ProxyCtx struct {
	Req  *http.Request
	Res  *http.Response
	Sess int64
}
