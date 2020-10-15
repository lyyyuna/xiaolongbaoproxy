package cmd

import (
	"github.com/spf13/cobra"
)

var basicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Start a basic http proxy with mitm",
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

}
