package patching_util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	// "github.com/sanity-io/litter"
)

type PatchManifest map[string]PlatformPatchManifest
type PlatformPatchManifest map[string]BranchPatchManifest
type BranchPatchManifest map[string]PatchInfo
type PatchInfo struct {
	Fixed    string `json:"fixed"`
	Original string `json:"original"`
	Patch    string `json:"patch"`
	PatchUrl string `json:"patch-url"`
}

func GetManifest(platform, branch string) (BranchPatchManifest, error) {
	var data PatchManifest
	// TODO figure out good CDN for this stuff?
	resp, err := http.Get("https://raw.githubusercontent.com/solsticegamestudios/GModCEFCodecFix/master/manifest.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error: received non-200 response code: %v", resp.StatusCode)

	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	platformManifest, exists := data[platform]
	if !exists {
		return nil, fmt.Errorf("Error: platform: \"%v\" not found in manifest.", platform)
	}
	branchManifest, exists := platformManifest[branch]
	if !exists {
		return nil, fmt.Errorf("Error: branch: \"%v\" not found in manifest.", branch)
	}
	// litter.Dump(branchManifest)
	return branchManifest, nil
}

func GetFileSHA256(filePath string) (string, error) {
	fileSHA256 := sha256.New()

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Couldn't open")
	}
	defer file.Close()

	buffer := make([]byte, 10485760)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("Something else")
		}
		if n == 0 {
			break
		}
		fileSHA256.Write(buffer[:n])
	}
	return fmt.Sprintf("%X", fileSHA256.Sum(nil)), nil
}
