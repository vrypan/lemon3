package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

const Dir = "lemon3"

const envXdgConfig = "XDG_CONFIG_HOME"

func dirExists(p string) (bool, error) {
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", p)
	}
	return true, nil
}

// ConfigDir returns the path to the configuration directory.
// Defaults to "~/.lemon3", or "$XDG_CONFIG_HOME/lemon3" if XDG_CONFIG_HOME is set.
// If both paths exist, it prioritizes "$XDG_CONFIG_HOME/lemon3".
// This function creates the path if it does not already exist.
//
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func ConfigDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	defaultPath := filepath.Join(usr.HomeDir, "."+Dir)
	defaultPathExists, err := dirExists(defaultPath)
	if err != nil {
		return "", err
	}

	xdgConfig, ok := os.LookupEnv(envXdgConfig)
	if ok && xdgConfig != "" {
		xdgConfigPath := filepath.Join(xdgConfig, Dir)
		xdgConfigPathExists, err := dirExists(xdgConfigPath)
		if err != nil {
			return "", err
		}

		if xdgConfigPathExists || !defaultPathExists {
			if !xdgConfigPathExists {
				if err := os.Mkdir(xdgConfigPath, 0755); err != nil {
					return "", fmt.Errorf("failed to create dir %s: %w", xdgConfigPath, err)
				}
			}
			return xdgConfigPath, nil
		}
	}

	if !defaultPathExists {
		if err := os.Mkdir(defaultPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create dir %s: %w", defaultPath, err)
		}
	}

	return defaultPath, nil
}
