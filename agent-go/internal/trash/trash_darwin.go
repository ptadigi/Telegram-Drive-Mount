//go:build darwin

package trash

import (
	"fmt"
	"os/exec"
	"strings"
)

func moveToTrash(path string) error {
	escaped := strings.ReplaceAll(path, `"`, `\"`)
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, escaped)
	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript trash: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
