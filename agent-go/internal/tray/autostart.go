package tray

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const autostartName = "TelegramVirtualDrive"

func EnableAutostart(execPath string) error {
	if execPath == "" {
		return errors.New("đường dẫn binary trống")
	}
	switch runtime.GOOS {
	case "windows":
		return enableAutostartWindows(execPath)
	case "darwin":
		return enableAutostartMac(execPath)
	default:
		return enableAutostartLinux(execPath)
	}
}

func DisableAutostart() error {
	switch runtime.GOOS {
	case "windows":
		return disableAutostartWindows()
	case "darwin":
		return disableAutostartMac()
	default:
		return disableAutostartLinux()
	}
}

func enableAutostartWindows(execPath string) error {
	cmd := exec.Command("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", autostartName, "/t", "REG_SZ", "/d", fmt.Sprintf("\"%s\" --tray", execPath), "/f")
	return cmd.Run()
}

func disableAutostartWindows() error {
	cmd := exec.Command("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", autostartName, "/f")
	return cmd.Run()
}

func enableAutostartMac(execPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	plist := strings.NewReplacer("__BIN__", execPath).Replace(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.tencuaban.telegramvirtualdrive</string>
  <key>ProgramArguments</key>
  <array>
    <string>__BIN__</string>
    <string>--tray</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><false/>
</dict>
</plist>`)
	return os.WriteFile(filepath.Join(dir, "com.tencuaban.telegramvirtualdrive.plist"), []byte(plist), 0o644)
}

func disableAutostartMac() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(home, "Library", "LaunchAgents", "com.tencuaban.telegramvirtualdrive.plist"))
}

func enableAutostartLinux(execPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "autostart")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	desktop := strings.NewReplacer("__BIN__", execPath).Replace(`[Desktop Entry]
Type=Application
Name=Ổ Đĩa Cloud Ảo
Exec="__BIN__" --tray
X-GNOME-Autostart-enabled=true
`)
	return os.WriteFile(filepath.Join(dir, "telegram-virtual-drive.desktop"), []byte(desktop), 0o644)
}

func disableAutostartLinux() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(home, ".config", "autostart", "telegram-virtual-drive.desktop"))
}
