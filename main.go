package main

import (
	"bytes"
	"fmt"
	"path"
	"sync"

	"gmod-cef-codec-fix-native/internal/patching_util"
	"gmod-cef-codec-fix-native/internal/steam_util"
	"gmod-cef-codec-fix-native/internal/ui"

	"github.com/sanity-io/litter"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	GMOD_APP_ID  = 4000
	GMOD_APP_DIR = "GarrysMod"
)

// Just compare the checksums as a demo for now
func process() {
	steamPath, err := steam_util.GetSteamPath()
	if err != nil {
		fmt.Println(err)
		return
	}

	lastSteamUser, err := steam_util.GetLastLoginUser(steamPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	litter.Dump(lastSteamUser)

	steamLibraries, err := steam_util.GetSteamLibraries(steamPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	gmodManifest, err := steam_util.GetGameManifest(steamLibraries, GMOD_APP_ID)
	if err != nil {
		fmt.Println(err)
		return
	}
	litter.Dump(gmodManifest)

	targetPlatform, err := steam_util.GetTargetPlatform(steamPath, GMOD_APP_ID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(targetPlatform)

	gmodAppInfo, err := steam_util.GetGameAppInfo(steamPath, GMOD_APP_ID)
	if err != nil {
		fmt.Println(err)
		return
	}
	litter.Dump(gmodAppInfo)

	gmodExeOptions, err := steam_util.GetGameLaunchOptions(steamPath, *lastSteamUser, GMOD_APP_ID)
	if err != nil {
		fmt.Println(err)
		return
	}
	litter.Dump(gmodExeOptions)

	gmodBranch := steam_util.GetGameBranch(gmodManifest)

	manifest, err := patching_util.GetManifest(targetPlatform, gmodBranch)
	if err != nil {
		fmt.Println(err)
		return
	}

	gmodGamePath, err := steam_util.FindGamePath(*steamLibraries, *lastSteamUser, GMOD_APP_DIR)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("GAME PATH", gmodGamePath)

	var wg sync.WaitGroup
	wg.Add(len(manifest))
	for filePath, patchInfo := range manifest {
		go func() {
			defer wg.Done()
			fileSha, err := patching_util.GetFileSHA256(path.Join(gmodGamePath, filePath))
			if err != nil {
				fmt.Println(err)
			}
			if fileSha == patchInfo.Fixed {
				fmt.Println(fmt.Sprintf("✅ %v", filePath))
			} else {
				fmt.Println(fmt.Sprintf("❌ %v", filePath))
			}
		}()
	}
	wg.Wait()
}

func main() {
	mainApp := app.New()
	mainWindow := mainApp.NewWindow("GmodCEFCodecFix-native demo")

	bgImage := canvas.NewImageFromReader(bytes.NewReader(ui.BgImgData), "bgImage")
	bgImage.FillMode = canvas.ImageFillContain

	textBox := ui.NewTransparentEntry()
	textBox.SetText("GmodCEFCodecFix-native demo\n(it only compares the checksums as a proof of concept)")
	textBox.Wrapping = fyne.TextWrapWord
	textBox.MultiLine = true

	var launchButton *widget.Button
	launchButton = widget.NewButton("Patch", func() {
		launchButton.Disable()
		go process()
	})
	launchButton.Importance = widget.HighImportance

	ui.AttachToConsole()
	ui.InterceptTextOutputToGui(textBox)

	mainWindowContent := container.NewBorder(
		// Top
		nil,

		// Bottom
		launchButton,

		// Left
		nil,

		// Right
		nil,

		// Center
		container.NewStack(
			container.New(&ui.BottomRightLayout{},
				bgImage,
			),
			textBox,
		),
	)
	mainWindow.SetContent(mainWindowContent)
	mainWindow.Resize(fyne.NewSize(900, 600))
	mainWindow.ShowAndRun()
}
