package tray

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func PickFolder(title string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return pickFolderWindows(title)
	case "darwin":
		return pickFolderMac(title)
	default:
		return pickFolderLinux(title)
	}
}

func pickFolderWindows(title string) (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description = '` + escapePS(title) + `'; if ($f.ShowDialog() -eq 'OK') { [Console]::Out.Write($f.SelectedPath) }`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", errors.New("hủy chọn thư mục")
	}
	return path, nil
}

func pickFolderMac(title string) (string, error) {
	script := `osascript -e 'POSIX path of (choose folder with prompt "` + escapeAS(title) + `")'`
	cmd := exec.Command("sh", "-c", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", errors.New("hủy chọn thư mục")
	}
	return path, nil
}

func pickFolderLinux(title string) (string, error) {
	cmd := exec.Command("zenity", "--file-selection", "--directory", "--title", title)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", errors.New("hủy chọn thư mục")
	}
	return path, nil
}

func escapePS(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func escapeAS(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}
