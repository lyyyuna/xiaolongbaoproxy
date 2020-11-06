package cmd

import (
	"fmt"
	"net/http"
	"xiaolongbaoproxy/pkg/proxy"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var mitmRecordCmd = &cobra.Command{
	Use:   "mitm-record",
	Short: "Start a mitm http proxy, and record HTTP request/response",
	Run:   runMitmProxyWithRecord,
}

func init() {
	mitmRecordCmd.Flags().StringVarP(&host, "server", "s", "0.0.0.0", "Specify the host server address.")
	mitmRecordCmd.Flags().IntVarP(&port, "port", "p", 8080, "Specify the port number.")
	mitmRecordCmd.Flags().StringVarP(&certpath, "certpath", "c", "root.crt", "Specify the path for the CA certificate.")
	mitmRecordCmd.Flags().StringVarP(&keypath, "keypath", "k", "root.key", "Specify the path for the CA private key.")
	mitmRecordCmd.Flags().StringVarP(&certcache, "certcache", "", "certstore.db", "Specify the path for the certificate cache store.")
}

func runMitmProxyWithRecord(cmd *cobra.Command, args []string) {
	addr := fmt.Sprintf("%v:%v", host, port)
	zap.S().Infof("Proxy server is hosting on %v", addr)

	p := proxy.NewMitmProxyServer(certpath, keypath, certcache, hook)
	http.ListenAndServe(addr, p)
}

func hook(ctx *proxy.ProxyCtx) {
	zap.S().Debugf("[%v] url is %v, %v", ctx.Session, ctx.Request.Host, ctx.Request.Url)
}
