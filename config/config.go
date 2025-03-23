package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var LEMON3_VERSION string

// Initialize configuration using Viper
func Load() string { // Load config and return config file path
	configDir, err := ConfigDir()
	if err != nil {
		log.Fatalf("Error getting config file: %v", err)
	}
	viper.SetEnvPrefix("LEMON3")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigName("config")
	viper.AddConfigPath(configDir)
	viper.SetConfigType("yaml")

	defaults := map[string]interface{}{
		"hub.host":    "hoyt.farcaster.xyz",
		"hub.port":    "2283",
		"hub.ssl":     "true",
		"key.public":  "",
		"key.private": "",
		"key.fid":     0,
		"host.addr":   "0.0.0.0:8080",
		"db.path":     "",
	}
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("Creating %s", filepath.Join(configDir, "config.yaml"))
			if err := viper.WriteConfigAs(filepath.Join(configDir, "config.yaml")); err != nil {
				log.Fatalf("Error writing config file: %v", err)
			}
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}

	downloadDir := viper.GetString("download.dir")
	if strings.HasPrefix(downloadDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting user home directory: %v", err)
		}
		viper.Set("download.dir", filepath.Join(home, downloadDir[1:]))
	}
	return viper.ConfigFileUsed()
}

var (
	GetString = viper.GetString
	GetInt    = viper.GetInt
	GetBool   = viper.GetBool
	BindPFlag = viper.BindPFlag
)
