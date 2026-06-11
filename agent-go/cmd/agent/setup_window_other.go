//go:build !windows || !tray

package main

// runSetupWindow is unavailable on non-Windows or non-tray builds; callers
// fall back to opening the system browser.
func runSetupWindow(url string) error {
	return errSetupWindowUnavailable
}
