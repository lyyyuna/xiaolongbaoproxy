package cmd

import (
	"github.com/spf13/cobra"
	"net/http"
	"xiaolongbaoproxy/internal"
)

var basicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Start a basic http proxy without mitm",
	Run:   runProxy,
}

func runProxy(cmd *cobra.Command, args []string) {
	p := internal.NewProxyHttpServer()
	http.ListenAndServe(":8080", p)
}
