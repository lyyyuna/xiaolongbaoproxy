package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "xiaolongbaoproxy",
	Short: "Start a HTTP/S proxy",
	Long:  "",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		log.Error(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	rootCmd.AddCommand(basicCmd)
	rootCmd.AddCommand(versionCmd)
}
