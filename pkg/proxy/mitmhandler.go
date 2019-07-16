package proxy

import (
	"bufio"
	"fmt"
	"github.com/google/logger"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"
)

type WrappedHandler struct {
	logg *logger.Logger
}

func (handler *WrappedHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method == "CONNECT" {
		host := req.Host
		matched, _ := regexp.MatchString(":[0-9]+$", host)
		if !matched {
			host += ":443"
		}
		connOut, err := net.DialTimeout("tcp", host, time.Second*30)
		if err != nil {
			handler.logg.Error("Dial out to ", host, " failed, the error is ", err)
			res.WriteHeader(502)
			return
		}
		defer connOut.Close()
		res.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		connIn, _, err := res.(http.Hijacker).Hijack()
		if err != nil {
			handler.logg.Error("Hijack http connection failed, the error is ", err)
			res.WriteHeader(502)
		}
		go io.Copy(connOut, connIn)
		io.Copy(connIn, connOut)
		return
	}

	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authentication")
	req.Header.Del("Proxy-Authorization")

	host := req.Host
	matched, _ := regexp.MatchString(":[0-9]+$", host)
	if !matched {
		host += ":80"
	}
	connOut, err := net.DialTimeout("tcp", host, time.Second*30)
	if err != nil {
		handler.logg.Error("Dial out to ", host, " failed, the error is ", err)
		res.WriteHeader(502)
		return
	}
	defer connOut.Close()

	// transfer client's request to remote server
	if err = req.Write(connOut); err != nil {
		handler.logg.Error("Fail to connect to remote server, the error is: ", err)
		res.WriteHeader(502)
		return
	}

	respFromRemote, err := http.ReadResponse(bufio.NewReader(connOut), req)
	if err != nil && err != io.EOF {
		handler.logg.Error("Fail to read response from remote server, the error is: ", err)
		res.WriteHeader(502)
	}
	defer respFromRemote.Body.Close()

	// writes respFromRemote back to client
	dump, err := httputil.DumpResponse(respFromRemote, true)
	if err != nil {
		handler.logg.Error("Fail to dump the response to bytes buffer, the error is: ", err)
		res.WriteHeader(502)
	}
	connHj, _, err := res.(http.Hijacker).Hijack()
	if err != nil {
		handler.logg.Error("Hijack fail to take over the TCP connection from client's request")
	}
	defer connHj.Close()
	_, err = connHj.Write(dump)
	if err != nil {
		handler.logg.Error("Fail to send response to clisne, the error is: ", err)
	}

	// dump the http response
	handler.dumpHTTP(req, respFromRemote)
}

func (handler *WrappedHandler) dumpHTTP(req *http.Request, res *http.Response) {
	for headerName, headerValue :=  range req.Header {
		fmt.Println(headerName, headerValue)
	}
	fmt.Println("===========")
	for headerName, headerValue := range res.Header {
		fmt.Println(headerName, headerValue)
	}
}