package steam_util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gmod-cef-codec-fix-native/internal/steam_steamid"
)

func GetGameBranch(manifest *VdfAppManifest) string {
	if manifest.AppState.UserConfig.BetaKey != "" {
		return manifest.AppState.UserConfig.BetaKey
	}
	return "main"
}

func GameIsInGoodState(manifest *VdfAppManifest) bool {
	if manifest.AppState.StateFlags != 4 || manifest.AppState.ScheduledAutoUpdate != 0 {
		return false
	}
	return true
}

func GameIsUsingProton(steamPath string, appId uint32) (bool, error) {
	steamConfig, err := GetConfig(steamPath, appId)
	if err != nil {
		return false, err
	}
	compatToolMap, exists := steamConfig.InstallConfigStore.Software.Valve.Steam.CompatToolMapping[appId]
	if exists {
		if strings.Contains(strings.ToLower(compatToolMap.Name), "proton") {
			// litter.Dump(gmodCompat)
			return true, nil
		}
	}
	// litter.Dump(steamConfig)
	return false, nil
}

func GetTargetPlatform(steamPath string, appId uint32) (string, error) {
	targetPlatform := runtime.GOOS
	if targetPlatform == "linux" {
		usingProton, err := GameIsUsingProton(steamPath, appId)
		if err != nil {
			return "", err
		}
		if usingProton {
			targetPlatform = "windows"
		}
	}
	// Use matching name from python sys.platform
	if targetPlatform == "windows" {
		targetPlatform = "win32"
	}
	return targetPlatform, nil
}

func GetGameLaunchOptions(steamPath string, steamUser SteamUser, appId uint32) (string, error) {
	localAppConfig, err := GetLocalConfig(steamPath, steamUser)
	// litter.Dump(localAppConfig)
	if err != nil {
		return "", err
	}
	// TODO warn about -nocromium
	// can it just be removed?
	if appConfig, exists := localAppConfig.UserLocalConfigStore.Software.Valve.Steam.Apps[appId]; exists {
		return appConfig.LaunchOptions, nil
	}
	return "", nil
}

func GetLastLoginUser(steamPath string) (*SteamUser, error) {
	loginUsers, err := GetLoginUsers(steamPath)
	if err != nil {
		return nil, err
	}

	timeStamp := 0
	var lastSteamUser *SteamUser
	for steamId64, steamUser := range loginUsers.Users {
		steamId, err := steam_steamid.NewSteamID(fmt.Sprintf("%v", steamId64))
		if err != nil {
			return nil, err
		}
		steamUser.SteamID64 = steamId64
		steamUser.AccountId = fmt.Sprintf("%v", steamId.AccountID)

		if steamUser.MostRecent == 1 {
			return &steamUser, nil
		} else if steamUser.Timestamp > timeStamp {
			lastSteamUser = &steamUser
			timeStamp = steamUser.Timestamp
		}
	}
	if lastSteamUser != nil {
		return lastSteamUser, nil
	}
	return nil, errors.New("Couldn't find last steam user")
}

func FindGamePath(steamLibraries VdfLibraryFolders, steamUser SteamUser, gameDirName string) (string, error) {
	for _, steamLib := range steamLibraries.Libraryfolders {
		for _, gamePath := range []string{
			filepath.Join(steamLib.Path, "steamapps", "common", gameDirName),
			filepath.Join(steamLib.Path, "steamapps", steamUser.AccountName, gameDirName),
		} {
			if stat, err := os.Stat(gamePath); err == nil && stat.IsDir() {
				// TODO warn about multiple installs
				return gamePath, nil
			}
		}
	}
	return "", errors.New("Couldn't get game path")
}

// Maybe useful to show the player's avatar in the GUI while this is running?
func GetUserAvatar(steamPath string, steamUser SteamUser) (string, error) {
	cachedAvatar := filepath.Join(steamPath, "config", "avatarcache", fmt.Sprintf("%v.png", steamUser.SteamID64))
	if stat, err := os.Stat(cachedAvatar); err == nil && !stat.IsDir() {
		return cachedAvatar, nil
	}
	return "", nil
}
