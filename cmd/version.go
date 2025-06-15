package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var LEMON3_VERSION string
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
