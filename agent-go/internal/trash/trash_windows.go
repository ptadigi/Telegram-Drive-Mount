//go:build windows

package trash

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	foDelete      = 0x0003
	fofAllowUndo  = 0x0040
	fofNoUI       = 0x0614 // FOF_NOCONFIRMATION|FOF_NOERRORUI|FOF_SILENT|FOF_NOCONFIRMMKDIR
	fofWantNuke   = 0x4000
)

type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var (
	modShell32         = syscall.NewLazyDLL("shell32.dll")
	procSHFileOperation = modShell32.NewProc("SHFileOperationW")
)

func moveToTrash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("chuẩn hoá đường dẫn: %w", err)
	}
	// Win32 SHFileOperation expects double-null terminated string.
	utf16, err := syscall.UTF16FromString(abs)
	if err != nil {
		return fmt.Errorf("encode đường dẫn UTF-16: %w", err)
	}
	utf16 = append(utf16, 0)
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &utf16[0],
		fFlags: fofAllowUndo | fofNoUI,
	}
	rc, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	if rc != 0 {
		return fmt.Errorf("SHFileOperation thất bại (mã %d)", rc)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("recycle bị huỷ")
	}
	return nil
}
