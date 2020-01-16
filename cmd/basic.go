package cmd

import (
	"github.com/spf13/cobra"
)

var basicCmd = &cobra.Command{
	Use:   "basic",
	Short: "Start a basic http proxy without mitm",
	Run:   nil,
}
