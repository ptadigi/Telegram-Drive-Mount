//go:build !tray

// Package tray exposes a no-op implementation when the agent is built without
// the systray frontend (CI default, headless VPS, etc.). The cmd/agent main
// imports tray unconditionally and switches behaviour through the --tray flag,
// so we still need a compiling Run + Hooks here.

package tray

import (
	"context"
	"errors"
)

// Hooks must mirror the real implementation so the caller can pass the same
// struct regardless of build tag.
type Hooks struct {
	BaseURL   string
	DataDir   string
	ExecPath  string
	OnPause   func()
	OnResume  func()
	OnAddRoot func(path string) error
	OnMount   func() (string, error)
	OnUnmount func() error
	OnQuit    func()
}

// Run is a no-op when the systray feature is not built in.
func Run(ctx context.Context, hooks Hooks) {
	if hooks.OnQuit != nil {
		// Block until ctx is cancelled, then signal quit so the agent shuts down
		// just like the real tray's Quit menu would.
		<-ctx.Done()
		hooks.OnQuit()
	}
}

// EnableAutostart returns ErrNotSupported so callers know the OS-level
// autostart hook is unavailable in this build.
func EnableAutostart(_ string) error { return errNotSupported }

// DisableAutostart mirrors EnableAutostart for symmetry.
func DisableAutostart() error { return errNotSupported }

// PickFolder returns an empty path; the caller should fall back to a manual
// input prompt in the PWA when this build does not include the tray.
func PickFolder(_ string) (string, error) { return "", errNotSupported }

var errNotSupported = errors.New("tray feature is disabled in this build (rebuild with -tags tray)")