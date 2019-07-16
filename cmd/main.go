package main

import (
	"flag"
	"fmt"
	"goproxy/pkg/proxy"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

func main() {
	port := flag.String("port", "8080", "Listening port")
	logpath := flag.String("log", "mitm.log", "Specify where to store the log")
	profile := flag.String("profile", "", "write cpu profile to file")

	flag.Parse()
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
			pprof.StopCPUProfile()
			os.Exit(1)
		}()
	}

	fmt.Println("The proxy is listening on port: ", *port)
	fmt.Println("Log will be written to: ", *logpath)

	server := proxy.StartProxy(*port, *logpath)
	server.ListenAndServe()
}