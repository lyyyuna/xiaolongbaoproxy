package proxy

import (
	"github.com/golang/glog"
	"net"
	"net/http"
	"time"
)

func StartProxy(port string) *http.Server{

	ips := getIPs()
	handler := &WrappedHandler{
		ips,
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: handler,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	glog.Info("Proxy server starting..")
	return server
}

func getIPs() (ips map[string]int) {
	ips = make(map[string]int)

	interfaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		panic("Fail to get interfaces' info.")
	}

	for _, addr := range interfaceAddrs {
		ip, ok := addr.(*net.IPNet)
		if ok && !ip.IP.IsLoopback() {
			ips[ip.IP.String()] = 1
		}
	}
	glog.Info("Local IP are: ", ips)

	return ips
}
