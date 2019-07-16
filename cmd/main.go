package main

import (
	"flag"
	"fmt"
	"goproxy/pkg/proxy"
)

func main() {
	port := flag.String("port", "8080", "Listening port")
	logpath := flag.String("log", "mitm.log", "Specify where to store the log")

	flag.Parse()

	fmt.Println("The proxy is listening on port: ", *port)
	fmt.Println("Log will be written to: ", *logpath)

	server := proxy.StartProxy(*port, *logpath)
	go server.ListenAndServe()
}
