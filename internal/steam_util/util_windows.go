package steam_util

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"strings"
)

func GetSteamPath() (string, error) {
	regKey, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("error opening registry key: %w", err)
	}
	defer regKey.Close()
	steamPath, _, err := regKey.GetStringValue("SteamPath")
	if err != nil {
		return "", fmt.Errorf("error querying SteamPath: %w", err)
	}
	steamPath = strings.ReplaceAll(steamPath, "/", "\\")
	return steamPath, nil
}
