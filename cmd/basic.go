package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
	"net/http"
	"xiaolongbaoproxy/internal"
)

var basicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Start a basic http proxy without mitm",
	Run:   runProxy,
}

var (
	host      string
	port      int
	mongoHost string
	mongoPort int
)

func init() {
	basicCmd.Flags().StringVarP(&host, "server", "s", "0.0.0.0", "Specify the host server address.")
	basicCmd.Flags().IntVarP(&port, "port", "p", 8080, "Specify the port number.")
	basicCmd.Flags().StringVarP(&mongoHost, "mongohost", "", "", "Specify the mongo server address.")
	basicCmd.Flags().IntVarP(&mongoPort, "mongoport", "", 27017, "Specify the mongo server port.")
}

func checkValidAddrPort(host string, port int) {
	if net.ParseIP(host) == nil {
		panic("IP address is invalid.")
	}

	if port < 1 || port > 65535 {
		panic("Invalid port address.")
	}
}

func checkValidMongoAddress(host string, port int) {
	if net.ParseIP(host) == nil {
		panic("IP address is invalid.")
	}

	if port < 1 || port > 65535 {
		panic("Invalid port address.")
	}
}

func runProxy(cmd *cobra.Command, args []string) {
	checkValidAddrPort(host, port)

	addr := fmt.Sprintf("%v:%v", host, port)
	log.Infof("Proxy server is hosting on %v", addr)
	p := internal.NewProxyHttpServer()
	http.ListenAndServe(addr, p)
}
