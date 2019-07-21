package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"goproxy/pkg/proxy"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

func main() {
	port := flag.String("port", "8080", "Listening port")
	profile := flag.String("profile", "", "write cpu profile to file")
	excludeIP := flag.String("excludeip", "", "exclude ip to forward")

	flag.Parse()
	defer glog.Flush()

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			fmt.Println("Profile error")
			return
		}
		pprof.StartCPUProfile(f)
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Println("exiting")
			glog.Flush()
			pprof.StopCPUProfile()
			os.Exit(1)
		}()
	}

	fmt.Println("The proxy is listening on port: ", *port)

	server := proxy.StartProxy(*port, *excludeIP)
	server.ListenAndServe()
}