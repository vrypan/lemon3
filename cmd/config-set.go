package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vrypan/lemon3/config"
)

var configsetCmd = &cobra.Command{
	Use:   "set [parameter] [value]",
	Short: "Set a config parameter",
	Long: `Example:
lemon3 config set ipfs.hub http://192.168.1.10:5001/api/v0
`,
	Run: config_set,
}

func config_set(cmd *cobra.Command, args []string) {
	config.Load()
	if len(args) != 2 {
		log.Fatal("Wrong number of arguments")
	}
	viper.Set(args[0], args[1])
	viper.WriteConfig()
}
func init() {
	configCmd.AddCommand(configsetCmd)
}
