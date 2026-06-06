//go:build !fuse

package vfs

import (
	"context"
	"errors"
	"runtime"

	"telegram-drive-agent/internal/drive"
)

type stubMounter struct{}

func newPlatformMounter(_ *drive.Service, _ string) Mounter { return nil }

func (stubMounter) Backend() string                                { return "none" }
func (stubMounter) Mount(_ context.Context, _ string) error        { return errors.New("mount engine không build kèm (cần build với -tags fuse)") }
func (stubMounter) Unmount() error                                 { return nil }
func (stubMounter) IsMounted() bool                                { return false }
func (stubMounter) MountPoint() string                             { return "" }

func defaultMountPoint() string {
	switch runtime.GOOS {
	case "windows":
		return "T:"
	case "darwin":
		return "/Volumes/Telegram Drive"
	default:
		return "~/TelegramDrive"
	}
}
