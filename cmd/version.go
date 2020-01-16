package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the xiaolongbao proxy",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("xiaolongbao proxy v3")
	},
}
