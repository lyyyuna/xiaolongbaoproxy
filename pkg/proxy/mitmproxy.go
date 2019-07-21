package proxy

import (
	"github.com/golang/glog"
	"net"
	"net/http"
	"strings"
	"time"
)

func StartProxy(port string, excludeIPs string) *http.Server{

	iparr := strings.Split(excludeIPs, ",")
	ips := make(map[string]int)
	for _, ip := range iparr {
		ips[ip] = 1
	}
	ips = getIPs(ips)
	handler := &WrappedHandler{
		ips,
	}
	server := &http.Server{
		Addr: ":" + port,
		Handler: handler,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	glog.Info("Proxy server starting..")
	return server
}

func getIPs(ips map[string]int) (map[string]int) {

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
