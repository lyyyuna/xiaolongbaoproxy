package cmd

import (
	"fmt"
	"net/http"
	"xiaolongbaoproxy/pkg/proxy"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var basicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Start a basic http proxy",
	Run:   runProxy,
}

var (
	host string
	port int
)

func init() {
	basicCmd.Flags().StringVarP(&host, "server", "s", "0.0.0.0", "Specify the host server address.")
	basicCmd.Flags().IntVarP(&port, "port", "p", 8080, "Specify the port number.")
}

func runProxy(cmd *cobra.Command, args []string) {
	addr := fmt.Sprintf("%v:%v", host, port)
	zap.S().Infof("Proxy server is hosting on %v", addr)

	p := proxy.NewProxyServer(nil)
	http.ListenAndServe(addr, p)
}
