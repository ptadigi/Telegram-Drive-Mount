//go:build tray

package tray

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/getlantern/systray"
)

type Hooks struct {
	BaseURL    string
	DataDir    string
	ExecPath   string
	OnPause    func()
	OnResume   func()
	OnAddRoot  func(path string) error
	OnMount    func() (string, error)
	OnUnmount  func() error
	OnQuit     func()
}

func Run(ctx context.Context, hooks Hooks) {
	systray.Run(func() {
		systray.SetTitle("á»” ÄÄ©a Cloud áº¢o")
		systray.SetTooltip("á»” ÄÄ©a Cloud áº¢o - Telegram backend")
		setIconBytes()
		open := systray.AddMenuItem("Má»Ÿ giao diá»‡n", "Má»Ÿ PWA trong trÃ¬nh duyá»‡t")
		dataDir := systray.AddMenuItem("Má»Ÿ thÆ° má»¥c dá»¯ liá»‡u", "Má»Ÿ Explorer/Finder táº¡i thÆ° má»¥c dá»¯ liá»‡u")
		addRoot := systray.AddMenuItem("ThÃªm thÆ° má»¥c Ä‘á»“ng bá»™", "Chá»n thÆ° má»¥c local Ä‘á»ƒ Ä‘á»“ng bá»™ Telegram")
		systray.AddSeparator()
		pause := systray.AddMenuItem("Táº¡m dá»«ng Ä‘á»“ng bá»™", "Táº¡m dá»«ng worker Ä‘á»“ng bá»™ Telegram")
		resume := systray.AddMenuItem("Tiáº¿p tá»¥c Ä‘á»“ng bá»™", "Báº­t láº¡i worker Ä‘á»“ng bá»™ Telegram")
		resume.Hide()
		systray.AddSeparator()
		mountItem := systray.AddMenuItem("Mount á»• áº£o", "Mount á»• Telegram Drive")
		unmountItem := systray.AddMenuItem("Unmount á»• áº£o", "ThÃ¡o á»• Telegram Drive")
		unmountItem.Hide()
		if hooks.OnMount == nil && hooks.OnUnmount == nil {
			mountItem.Hide()
		}
		systray.AddSeparator()
		autostart := systray.AddMenuItemCheckbox("Tá»± khá»Ÿi Ä‘á»™ng cÃ¹ng mÃ¡y", "KÃ­ch hoáº¡t khi Ä‘Äƒng nháº­p OS", false)
		systray.AddSeparator()
		quit := systray.AddMenuItem("ThoÃ¡t", "Táº¯t Agent vÃ  thoÃ¡t")

		go func() {
			for {
				select {
				case <-ctx.Done():
					systray.Quit()
					return
				case <-open.ClickedCh:
					_ = openURL(hooks.BaseURL)
				case <-dataDir.ClickedCh:
					_ = openPath(hooks.DataDir)
				case <-addRoot.ClickedCh:
					if hooks.OnAddRoot != nil {
						if path, err := PickFolder("Chá»n thÆ° má»¥c Ä‘á»“ng bá»™ Telegram"); err == nil {
							_ = hooks.OnAddRoot(path)
						}
					}
				case <-pause.ClickedCh:
					if hooks.OnPause != nil {
						hooks.OnPause()
					}
					pause.Hide()
					resume.Show()
				case <-resume.ClickedCh:
					if hooks.OnResume != nil {
						hooks.OnResume()
					}
					resume.Hide()
					pause.Show()
				case <-mountItem.ClickedCh:
					if hooks.OnMount != nil {
						if _, err := hooks.OnMount(); err == nil {
							mountItem.Hide()
							unmountItem.Show()
						}
					}
				case <-unmountItem.ClickedCh:
					if hooks.OnUnmount != nil {
						if err := hooks.OnUnmount(); err == nil {
							unmountItem.Hide()
							mountItem.Show()
						}
					}
				case <-autostart.ClickedCh:
					if autostart.Checked() {
						_ = DisableAutostart()
						autostart.Uncheck()
					} else {
						_ = EnableAutostart(hooks.ExecPath)
						autostart.Check()
					}
				case <-quit.ClickedCh:
					if hooks.OnQuit != nil {
						hooks.OnQuit()
					}
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
}

func setIconBytes() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(iconICO)
		return
	}
	systray.SetIcon(iconPNG)
}

func openURL(url string) error {
	if url == "" {
		return fmt.Errorf("base url trá»‘ng")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func openPath(path string) error {
	if path == "" {
		return fmt.Errorf("Ä‘Æ°á»ng dáº«n trá»‘ng")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", filepath.Clean(path)).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

