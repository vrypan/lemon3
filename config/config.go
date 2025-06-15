package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Initialize configuration using Viper
func Load() string { // Load config and return config file path
	configDir, _ := ConfigDir()

	viper.SetEnvPrefix("LEMON3") // LEMON3_ env vars cna override config.
	// For example, you can set LEMON3_HUB_HOST
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(fmt.Sprintf("%s%c%s", configDir, os.PathSeparator, "config.yaml"))
	viper.ReadInConfig()
	return viper.ConfigFileUsed()
}

var (
	GetString = viper.GetString
	GetInt    = viper.GetInt
	GetBool   = viper.GetBool
	BindPFlag = viper.BindPFlag
)
