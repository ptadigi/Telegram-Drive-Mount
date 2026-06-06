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
		systray.SetTitle("Ổ Đĩa Cloud Ảo")
		systray.SetTooltip("Ổ Đĩa Cloud Ảo - Telegram backend")
		setIconBytes()
		open := systray.AddMenuItem("Mở giao diện", "Mở PWA trong trình duyệt")
		dataDir := systray.AddMenuItem("Mở thư mục dữ liệu", "Mở Explorer/Finder tại thư mục dữ liệu")
		addRoot := systray.AddMenuItem("Thêm thư mục đồng bộ", "Chọn thư mục local để đồng bộ Telegram")
		systray.AddSeparator()
		pause := systray.AddMenuItem("Tạm dừng đồng bộ", "Tạm dừng worker đồng bộ Telegram")
		resume := systray.AddMenuItem("Tiếp tục đồng bộ", "Bật lại worker đồng bộ Telegram")
		resume.Hide()
		systray.AddSeparator()
		mountItem := systray.AddMenuItem("Mount ổ ảo", "Mount ổ Telegram Drive")
		unmountItem := systray.AddMenuItem("Unmount ổ ảo", "Tháo ổ Telegram Drive")
		unmountItem.Hide()
		if hooks.OnMount == nil && hooks.OnUnmount == nil {
			mountItem.Hide()
		}
		systray.AddSeparator()
		autostart := systray.AddMenuItemCheckbox("Tự khởi động cùng máy", "Kích hoạt khi đăng nhập OS", false)
		systray.AddSeparator()
		quit := systray.AddMenuItem("Thoát", "Tắt Agent và thoát")

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
						if path, err := PickFolder("Chọn thư mục đồng bộ Telegram"); err == nil {
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
	systray.SetTemplateIcon(iconBytes, iconBytes)
}

func openURL(url string) error {
	if url == "" {
		return fmt.Errorf("base url trống")
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
		return fmt.Errorf("đường dẫn trống")
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

var iconBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0xF3, 0xFF, 0x61, 0x00, 0x00, 0x00,
	0x4F, 0x49, 0x44, 0x41, 0x54, 0x38, 0x4F, 0x63, 0x60, 0x18, 0x05, 0xA3,
	0x60, 0x14, 0x8C, 0x82, 0x51, 0x30, 0x0A, 0x46, 0xC1, 0x28, 0x18, 0x05,
	0xA3, 0x60, 0x14, 0x8C, 0x82, 0x51, 0x30, 0x0A, 0x46, 0xC1, 0x28, 0x18,
	0x05, 0xA3, 0x60, 0x14, 0x8C, 0x82, 0x51, 0x30, 0x0A, 0x46, 0xC1, 0x28,
	0x18, 0x05, 0xA3, 0x60, 0x14, 0x8C, 0x82, 0x51, 0x30, 0x0A, 0x46, 0xC1,
	0x28, 0x18, 0x05, 0xA3, 0x60, 0x14, 0x8C, 0x82, 0x51, 0x30, 0x0A, 0x46,
	0xC1, 0x28, 0x18, 0x05, 0xA3, 0x60, 0x14, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}
