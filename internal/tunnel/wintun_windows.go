//go:build windows

package tunnel

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed wintun
var wintunFS embed.FS

func setupWintun() (string, error) {
	var path string
	switch runtime.GOARCH {
	case "amd64":
		path = "wintun/amd64/wintun.dll"
	case "arm64":
		path = "wintun/arm64/wintun.dll"
	case "arm":
		path = "wintun/arm/wintun.dll"
	case "386":
		path = "wintun/x86/wintun.dll"
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	data, err := wintunFS.ReadFile(path)
	if err != nil {
		return "", err
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dllPath := filepath.Join(filepath.Dir(exe), "wintun.dll")
	if err := os.WriteFile(dllPath, data, 0644); err != nil {
		return "", err
	}
	return dllPath, nil
}
