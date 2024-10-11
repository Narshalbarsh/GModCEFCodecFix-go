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
	steamPath := filepath.Join(homeDir, "Library", "Application Support", "Steam")
	if stat, err := os.Stat(steamPath); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("steam directory not found at %s", steamPath)
	}
	return steamPath, nil
}
