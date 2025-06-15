package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vrypan/lemon3/config"
)

type ConfigEntry struct {
	Key         string
	Default     string
	Description string
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure lemon3",
	Run: func(cmd *cobra.Command, args []string) {
		configs := []ConfigEntry{
			{
				Key:         "farcaster.node",
				Default:     "hub.merv.fun:3383",
				Description: "Farcaster node, in host:port format",
			},
			{
				Key:         "farcaster.ssl",
				Default:     "false",
				Description: "Use SSL? Enter 'true' or 'false'",
			},
			{
				Key:         "farcaster.fname",
				Default:     "",
				Description: "Your Farcaster username (fname)",
			},
			{
				Key:         "farcaster.appkey",
				Default:     "",
				Description: "App key used to authenticate with the Farcaster Hub.\nYou can create one at https://www.castkeys.xyz",
			},
			{
				Key:         "ipfs.hub",
				Default:     "http://127.0.0.1:5001/api/v0",
				Description: "IPFS API endpoint (usually your local Kubo node)",
			},
		}

		configFile := config.Load()
		reader := bufio.NewReader(os.Stdin)
		for _, entry := range configs {
			viper.SetDefault(entry.Key, entry.Default)

			fmt.Println()
			fmt.Printf("%s\n", entry.Description)
			value := entry.Default
			if v := config.GetString(entry.Key); v != "" {
				value = v
			}
			fmt.Printf("%s [%s]: ", entry.Key, value)

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				viper.Set(entry.Key, entry.Default)
			} else {
				viper.Set(entry.Key, input)
			}
		}

		// Save configuration
		fmt.Printf("\nSaving configuration to %s...\n", configFile)
		if err := viper.WriteConfigAs(configFile); err != nil {
			fmt.Println("Error writing config:", err)
		} else {
			fmt.Println("Configuration saved.")
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
