package proxy

import (
	"bufio"
	"github.com/golang/glog"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"
)

type WrappedHandler struct { }

var cnt int64

func (handler *WrappedHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	// do some statistics
	handler.statistics()

	if req.Method == "CONNECT" {
		host := req.Host
		matched, _ := regexp.MatchString(":[0-9]+$", host)
		if !matched {
			host += ":443"
		}
		connOut, err := net.DialTimeout("tcp", host, time.Second*30)
		if err != nil {
			glog.Error("Dial out to ", host, " failed, the error is ", err)
			res.WriteHeader(502)
			return
		}
		defer connOut.Close()
		res.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		connIn, _, err := res.(http.Hijacker).Hijack()
		if err != nil {
			glog.Error("Hijack http connection failed, the error is ", err)
			res.WriteHeader(502)
			return
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
		glog.Error("Dial out to ", host, " failed, the error is ", err)
		res.WriteHeader(502)
		return
	}
	defer connOut.Close()

	// transfer client's request to remote server
	if err = req.Write(connOut); err != nil {
		glog.Error("Fail to connect to remote server, the error is: ", err)
		res.WriteHeader(502)
		return
	}

	respFromRemote, err := http.ReadResponse(bufio.NewReader(connOut), req)
	if err != nil && err != io.EOF {
		glog.Error("Fail to read response from remote server, the error is: ", err)
		res.WriteHeader(502)
		return
	}
	defer respFromRemote.Body.Close()

	// writes respFromRemote back to client
	dump, err := httputil.DumpResponse(respFromRemote, true)
	if err != nil {
		glog.Error("Fail to dump the response to bytes buffer, the error is: ", err)
		res.WriteHeader(502)
		return
	}
	connHj, _, err := res.(http.Hijacker).Hijack()
	if err != nil {
		glog.Error("Hijack fail to take over the TCP connection from client's request")
		res.WriteHeader(502)
		return
	}
	defer connHj.Close()
	_, err = connHj.Write(dump)
	if err != nil {
		glog.Error("Fail to send response to clisne, the error is: ", err)
		res.WriteHeader(502)
		return
	}

	// dump the http response
	handler.dumpHTTP(req, respFromRemote)
}

func (handler *WrappedHandler) dumpHTTP(req *http.Request, res *http.Response) {
	/*for headerName, headerValue :=  range req.Header {
		fmt.Println(headerName, headerValue)
	}
	fmt.Println("===========")
	for headerName, headerValue := range res.Header {
		fmt.Println(headerName, headerValue)
	}*/
}

func (handler *WrappedHandler) statistics() {
	cnt++
	if cnt % 10 == 0 {
		glog.Info("Has processed requests: ", cnt)
	}
}