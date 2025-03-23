package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vrypan/lemon3/config"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.LEMON3_VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
