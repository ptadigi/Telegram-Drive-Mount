//go:build linux

package trash

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func moveToTrash(path string) error {
	if _, err := exec.LookPath("gio"); err == nil {
		cmd := exec.Command("gio", "trash", "--", path)
		if out, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else {
			fmt.Printf("gio trash fallback: %v %s\n", err, strings.TrimSpace(string(out)))
		}
	}
	return xdgTrash(path)
}

func xdgTrash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	trashDir := filepath.Join(home, ".local", "share", "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return err
	}
	base := filepath.Base(abs)
	target := filepath.Join(filesDir, base)
	suffix := 1
	for {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		}
		target = filepath.Join(filesDir, base+"."+strconv.Itoa(suffix))
		suffix++
	}
	if err := os.Rename(abs, target); err != nil {
		return err
	}
	infoFile := filepath.Join(infoDir, filepath.Base(target)+".trashinfo")
	content := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", abs, time.Now().Format("2006-01-02T15:04:05"))
	return os.WriteFile(infoFile, []byte(content), 0o600)
}
