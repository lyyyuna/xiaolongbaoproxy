package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			s := strings.Split(f.Function, ".")
			funcname := s[len(s)-1]
			_, filename := path.Split(f.File)
			line := strconv.Itoa(f.Line)
			return "[" + funcname + "]", "[" + filename + ":" + line + "]"
		},
		ForceColors: true,
	})
	log.SetReportCaller(true)
	rootCmd.AddCommand(basicCmd)
	rootCmd.AddCommand(versionCmd)
}
