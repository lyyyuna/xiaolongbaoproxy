package proxy

import (
	"bufio"
	"github.com/golang/glog"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"
)

type WrappedHandler struct {
	localIP map[string]int
}

var cnt int64
var blacklist = map[string]int {
"www.google.com" : 1,
"twitter.com" : 1,
"www.instagram.com" : 1,
"m.youtube.com" : 1,
"www.youtube.com" : 1,
"mobile.twitter.com" : 1,
}

func (handler *WrappedHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	// do some statistics
	handler.statistics()

	// check if local
	if handler.checkIfLocalDest(strings.Split(req.Host, ":")[0]) {
		glog.Error("The destination is a self, maybe attack")
		res.WriteHeader(502)
		return
	}

	// check if in the blacklist
	if handler.checkIfBlacklist(strings.Split(req.Host, ":")[0]) {
		glog.Error("The host is in the blacklist: ", req.Host)
		res.WriteHeader(502)
		return
	}

	if req.Method == "CONNECT" {
		host := req.Host
		matched, _ := regexp.MatchString(":[0-9]+$", host)
		if !matched {
			host += ":443"
		}
		connOut, err := net.DialTimeout("tcp", host, time.Second*5)
		if err != nil {
			glog.Error("Dial out to ", host, " failed, the error is ", err)
			res.WriteHeader(502)
			return
		}
		defer connOut.Close()
		// check if local  by real dns' ip
		if handler.checkIfLocalDest(strings.Split(connOut.RemoteAddr().String(), ":")[0]) {
			glog.Error("The destination is a self, maybe attack")
			res.WriteHeader(502)
			return
		}
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
	connOut, err := net.DialTimeout("tcp", host, time.Second*5)
	if err != nil {
		glog.Error("Dial out to ", host, " failed, the error is ", err)
		res.WriteHeader(502)
		return
	}
	defer connOut.Close()

	// check if local by real dns' ip
	if handler.checkIfLocalDest(strings.Split(connOut.RemoteAddr().String(), ":")[0]) {
		glog.Error("The destination is a self, maybe attack")
		res.WriteHeader(502)
		return
	}

	// transfer client's request to remote server
	connOut.SetDeadline(time.Now().Add(time.Second * 5))
	if err = req.Write(connOut); err != nil {
		glog.Error("Fail to send client's data to remote server, the error is: ", err)
		res.WriteHeader(502)
		return
	}

	connOut.SetDeadline(time.Now().Add(time.Second * 5))
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
		glog.Info("MONITOR: Has processed requests: ", cnt)
	}
}

func (handler *WrappedHandler) checkIfLocalDest(dest string) bool {
	if _, ok := handler.localIP[dest]; ok {
		return true
	}

	return false
}

func (handler *WrappedHandler) checkIfBlacklist(host string) bool {
	_, ok := blacklist[host]
	if ok {
		return true
	}
	return false
}