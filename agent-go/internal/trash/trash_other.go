//go:build !windows && !darwin && !linux

package trash

func moveToTrash(_ string) error {
	return ErrNotSupported
}
