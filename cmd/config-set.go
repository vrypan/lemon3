package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configsetCmd = &cobra.Command{
	Use:   "set parameter value",
	Short: "Set a config parameter",
	Long: `Examples:
lemon3 config set hub.host 192.168.1.1
lemon3 config set hub.port 2283
lemon3 config set hub.ssl false`,
	Run: config_set,
}

func config_set(cmd *cobra.Command, args []string) {
	//config.Load()
	if len(args) != 2 {
		log.Fatal("Wrong number of arguments")
	}
	viper.Set(args[0], args[1])
	viper.WriteConfig()
}
func init() {
	configCmd.AddCommand(configsetCmd)
}
