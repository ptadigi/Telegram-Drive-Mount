//go:build windows && tray

package main

import (
	"github.com/jchv/go-webview2"
)

// runSetupWindow opens a native WebView2 window pointed at the local /setup
// page. Runs in its own process (spawned via --setup-window) so it doesn't
// fight the systray for the main UI thread.
func runSetupWindow(url string) error {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug: false,
		WindowOptions: webview2.WindowOptions{
			Title:  "Ổ Đĩa Cloud Ảo - Thiết lập",
			Width:  640,
			Height: 760,
			IconId: 0,
			Center: true,
		},
	})
	if w == nil {
		return errSetupWindowUnavailable
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	return nil
}
