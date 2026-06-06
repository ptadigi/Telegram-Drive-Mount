//go:build darwin

package trash

import (
	"fmt"
	"os/exec"
	"strings"
)

func moveToTrash(path string) error {
	cmd := exec.Command("osascript", "-e", `on run argv
		tell application "Finder" to delete POSIX file (item 1 of argv)
	end run`, "--", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript trash: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
