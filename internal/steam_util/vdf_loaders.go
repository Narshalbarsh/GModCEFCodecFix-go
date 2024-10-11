package steam_util

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gmod-cef-codec-fix-native/internal/steam_appcache"
)

func GetSteamLibraries(steamPath string) (*VdfLibraryFolders, error) {
	var steamLibraryFolders VdfLibraryFolders
	err := initVdfStructFromFile(
		filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"),
		&steamLibraryFolders,
	)
	if err != nil {
		return nil, err
	}
	// litter.Dump(steamLibraryFolders)
	return &steamLibraryFolders, nil
}

func GetLoginUsers(steamPath string) (*VdfLoginUsers, error) {
	var loginUsers VdfLoginUsers
	err := initVdfStructFromFile(
		filepath.Join(steamPath, "config", "loginusers.vdf"),
		&loginUsers,
	)
	if err != nil {
		return nil, err
	}
	// litter.Dump(steamLibraryFolders)
	return &loginUsers, nil
}

func GetGameManifest(steamLibraries *VdfLibraryFolders, appId uint32) (*VdfAppManifest, error) {
	for _, steamLib := range steamLibraries.Libraryfolders {
		var steamGameManifest VdfAppManifest
		err := initVdfStructFromFile(
			filepath.Join(steamLib.Path, "steamapps", fmt.Sprintf("appmanifest_%v.acf", appId)),
			&steamGameManifest,
		)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			continue
		}
		// litter.Dump(steamGameManifest)
		return &steamGameManifest, nil
	}
	return nil, errors.New(fmt.Sprintf("Couldn't parse any game manifest"))
}

func GetGameAppInfo(steamPath string, appId uint32) (*VdfAppInfo, error) {
	vdfFilePath := path.Join(steamPath, "appcache", "appinfo.vdf")
	var appInfo VdfAppInfo
	vdfFile, err := os.Open(vdfFilePath)
	defer vdfFile.Close()
	if err != nil {
		fmt.Print(err)
	}
	app, err := steam_appcache.GetGameSpecificAppInfo(vdfFile, appId)
	if err != nil {
		return nil, err
	}
	err = populateStructFromMap(app, &appInfo)
	if err != nil {
		return nil, err
	}
	// litter.Dump(appInfo)
	return &appInfo, nil
}

func GetConfig(steamPath string, appId uint32) (*VdfConfig, error) {
	var steamConfig VdfConfig
	err := initVdfStructFromFile(
		filepath.Join(steamPath, "config", "config.vdf"),
		&steamConfig,
	)
	if err != nil {
		return nil, err
	}
	return &steamConfig, nil
}

func GetLocalConfig(steamPath string, steamUser SteamUser) (*VdfLocalConfig, error) {
	var localAppConfig VdfLocalConfig
	err := initVdfStructFromFile(
		filepath.Join(steamPath, "userdata", steamUser.AccountId, "config", "localconfig.vdf"),
		&localAppConfig,
	)
	if err != nil {
		return nil, err
	}
	return &localAppConfig, nil
}
