package cmd

import (
	"fmt"
	"net/http"
	"xiaolongbaoproxy/pkg/proxy"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var mitmCmd = &cobra.Command{
	Use:   "mitm",
	Short: "Start a mitm http proxy",
	Run:   runMitmProxy,
}

var (
	certpath string
	keypath  string
)

func init() {
	mitmCmd.Flags().StringVarP(&host, "server", "s", "0.0.0.0", "Specify the host server address.")
	mitmCmd.Flags().IntVarP(&port, "port", "p", 8080, "Specify the port number.")
	mitmCmd.Flags().StringVarP(&certpath, "certpath", "c", "root.crt", "Specify the path for the CA certificate.")
	mitmCmd.Flags().StringVarP(&keypath, "keypath", "k", "root.key", "Specify the path for the CA private key.")
}

func runMitmProxy(cmd *cobra.Command, args []string) {
	addr := fmt.Sprintf("%v:%v", host, port)
	zap.S().Infof("Proxy server is hosting on %v", addr)

	p := proxy.NewMitmProxyServer(certpath, keypath, nil)
	http.ListenAndServe(addr, p)
}
