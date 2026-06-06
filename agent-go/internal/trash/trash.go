// Package trash moves files into the OS Recycle Bin / Trash with undo support.
package trash

import "errors"

// ErrNotSupported is returned when the current platform has no recycle implementation.
var ErrNotSupported = errors.New("recycle bin không hỗ trợ trên platform này")

// MoveToTrash moves the file/directory at path into the OS recycle bin.
// On unsupported platforms it returns ErrNotSupported.
func MoveToTrash(path string) error {
	if path == "" {
		return errors.New("đường dẫn trống")
	}
	return moveToTrash(path)
}
