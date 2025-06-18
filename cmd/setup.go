package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func getDesktopPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // last resort could be current directory
	}

	// Default guess
	desktop := filepath.Join(home, "Desktop")

	// For Linux: check if user-dirs.dirs exists and parse it
	if runtime.GOOS == "linux" {
		xdgConfig := filepath.Join(home, ".config", "user-dirs.dirs")
		if f, err := os.ReadFile(xdgConfig); err == nil {
			lines := string(f)
			for _, line := range strings.Split(lines, "\n") {
				if strings.HasPrefix(line, "XDG_DESKTOP_DIR") {
					// Format: XDG_DESKTOP_DIR="$HOME/Desktop"
					start := strings.Index(line, "\"")
					end := strings.LastIndex(line, "\"")
					if start >= 0 && end > start {
						path := line[start+1 : end]
						path = strings.ReplaceAll(path, "$HOME", home)
						return path
					}
				}
			}
		}
	}
	// Check if the path exists
	if _, err := os.Stat(desktop); os.IsNotExist(err) {
		return home // use home directory if Desktop doesn't exist
	}
	return desktop
}

func defaultDownloadDir() string {
	return filepath.Join(getDesktopPath(), "lemon3")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure lemon3",
	Run: func(cmd *cobra.Command, args []string) {

		configs := []ConfigEntry{
			{
				Key:     "farcaster.node.address",
				Default: "",
				Description: `Farcaster node, in host:port format. Port is usually 3383
To use Neynar nodes, check out the instructions at https://github.com/vrypan/lemon3`,
			},
			{
				Key:         "farcaster.node.ssl",
				Default:     "false",
				Description: "Use SSL? Enter 'true' or 'false'",
			},
			{
				Key:         "farcaster.node.apikey",
				Default:     "",
				Description: "If you use a hub provided by Neynar or similar services, enter your API key",
			},
			{
				Key:         "farcaster.account.fname",
				Default:     "",
				Description: "Your Farcaster username (fname)",
			},
			{
				Key:         "farcaster.account.appkey",
				Default:     "",
				Description: "App key used to authenticate with the Farcaster Hub.\nYou can create one at https://www.castkeys.xyz",
			},
			{
				Key:         "ipfs.hub",
				Default:     "http://127.0.0.1:5001/api/v0",
				Description: "IPFS API endpoint (usually your local Kubo node)",
			},
			{
				Key:         "download.dir",
				Default:     defaultDownloadDir(),
				Description: "Directory where downloads will be saved",
			},
		}

		configFile := config.Load()
		reader := bufio.NewReader(os.Stdin)
		for _, entry := range configs {
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
				viper.Set(entry.Key, value)
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
