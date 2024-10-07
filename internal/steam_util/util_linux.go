package steam_util

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetSteamPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	steamPaths := []string{
		filepath.Join(homeDir, "snap", "steam", "common", ".local", "share", "Steam"),
		filepath.Join(homeDir, ".steam", "steam"),
		filepath.Join(homeDir, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		filepath.Join(os.Getenv("XDG_DATA_HOME"), "Steam"),
	}
	for _, steamPath := range steamPaths {
		if stat, err := os.Stat(steamPath); err == nil && stat.IsDir() {
			// TODO warn about multiple installs
			return steamPath, nil
		}
	}
	return "", fmt.Errorf("steam directory not found in any known locations")
}
