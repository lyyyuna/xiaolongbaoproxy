package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "xiaolongbaoproxy",
	Short: "Start a HTTP/S proxy",
	Long:  "",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		zap.L().Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	rootCmd.AddCommand(basicCmd)
	rootCmd.AddCommand(versionCmd)
}
