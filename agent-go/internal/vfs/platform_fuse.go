//go:build fuse

package vfs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/winfsp/cgofuse/fuse"

	"telegram-drive-agent/internal/drive"
)

type fuseMounter struct {
	host    *fuse.FileSystemHost
	fs      *driveFS
	mu      sync.Mutex
	mounted atomic.Bool
	point   string
}

func newPlatformMounter(backend Backend, dataDir string) Mounter {
	if backend == nil {
		return nil
	}
	fs := newDriveFS(backend, dataDir)
	host := fuse.NewFileSystemHost(fs)
	host.SetCapReaddirPlus(true)
	host.SetCapCaseInsensitive(false)
	return &fuseMounter{host: host, fs: fs}
}

func (m *fuseMounter) Backend() string {
	switch runtime.GOOS {
	case "windows":
		return "winfsp"
	case "darwin":
		return "fuse-t"
	default:
		return "fuse"
	}
}

func (m *fuseMounter) Mount(ctx context.Context, mountPoint string) error {
	m.mu.Lock()
	if m.mounted.Load() {
		m.mu.Unlock()
		return errors.New("ổ ảo đã được mount")
	}
	m.point = mountPoint
	m.mu.Unlock()

	options := []string{"-o", "volname=TelegramDrive"}
	if runtime.GOOS == "windows" {
		options = []string{}
	}
	go func() {
		<-ctx.Done()
		m.host.Unmount()
		m.mounted.Store(false)
	}()
	// Optimistic flag: mark as mounted before the blocking call so that callers
	// observing status quickly after Mount() see mounted=true. cgofuse's Mount
	// blocks for the lifetime of the filesystem, so we cannot rely on the
	// return value to flip the flag.
	m.mounted.Store(true)
	if !m.host.Mount(mountPoint, options) {
		m.mounted.Store(false)
		return errors.New("không mount được ổ ảo, kiểm tra WinFsp/FUSE")
	}
	m.mounted.Store(false)
	return nil
}

func (m *fuseMounter) Unmount() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.mounted.Load() {
		return nil
	}
	if !m.host.Unmount() {
		return errors.New("không unmount được ổ ảo")
	}
	m.mounted.Store(false)
	return nil
}

func (m *fuseMounter) IsMounted() bool   { return m.mounted.Load() }
func (m *fuseMounter) MountPoint() string { return m.point }

func defaultMountPoint() string {
	switch runtime.GOOS {
	case "windows":
		return "T:"
	case "darwin":
		return "/Volumes/Telegram Drive"
	default:
		return filepath.Join("/tmp", "telegram-drive")
	}
}

// driveFS adapts drive.Service to a read-write FUSE filesystem with write-back to Telegram.
type driveFS struct {
	fuse.FileSystemBase
	svc      Backend
	dataDir  string
	mu       sync.Mutex
	handles  map[uint64]*writeHandle
	nextHand uint64
	pending  map[string]*writeHandle // key: lowercased mount path "/foo/bar.txt"
}

type writeHandle struct {
	parentID string
	parent   string
	name     string
	tempPath string
	file     *os.File
	dirty    bool
	size     int64
}

func newDriveFS(backend Backend, dataDir string) *driveFS {
	return &driveFS{svc: backend, dataDir: dataDir, handles: map[uint64]*writeHandle{}, pending: map[string]*writeHandle{}}
}

func pendingKey(p string) string {
	return strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
}

func (d *driveFS) tempDir() string {
	dir := filepath.Join(d.dataDir, "fuse-tmp")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func (d *driveFS) Open(path string, flags int) (int, uint64) {
	return 0, 0
}

func (d *driveFS) Statfs(path string, stat *fuse.Statfs_t) int {
	const blockSize = 4096
	const totalBytes = 1 << 50 // 1 PiB virtual capacity
	stat.Bsize = blockSize
	stat.Frsize = blockSize
	stat.Blocks = totalBytes / blockSize
	stat.Bfree = stat.Blocks / 2
	stat.Bavail = stat.Bfree
	stat.Files = 1 << 20
	stat.Ffree = stat.Files / 2
	stat.Namemax = 255
	return 0
}

func (d *driveFS) Create(path string, flags int, mode uint32) (int, uint64) {
	parent, leaf := splitPath(path)
	if leaf == "" {
		return -fuse.EINVAL, 0
	}
	parentID, err := d.svc.ResolveFolderByPath(context.Background(), parent)
	if err != nil {
		return -fuse.ENOENT, 0
	}
	tempPath := filepath.Join(d.tempDir(), randomTempName(leaf))
	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		return -fuse.EIO, 0
	}
	d.mu.Lock()
	d.nextHand++
	hid := d.nextHand
	h := &writeHandle{parentID: parentID, parent: parent, name: leaf, tempPath: tempPath, file: f, dirty: true}
	d.handles[hid] = h
	d.pending[pendingKey(path)] = h
	d.mu.Unlock()
	return 0, hid
}

func (d *driveFS) Write(path string, buff []byte, ofst int64, fh uint64) int {
	d.mu.Lock()
	h := d.handles[fh]
	d.mu.Unlock()
	if h == nil {
		return -fuse.EBADF
	}
	if _, err := h.file.WriteAt(buff, ofst); err != nil {
		return -fuse.EIO
	}
	if end := ofst + int64(len(buff)); end > h.size {
		h.size = end
	}
	h.dirty = true
	return len(buff)
}

func (d *driveFS) Truncate(path string, size int64, fh uint64) int {
	d.mu.Lock()
	h := d.handles[fh]
	d.mu.Unlock()
	if h != nil {
		if err := h.file.Truncate(size); err != nil {
			return -fuse.EIO
		}
		h.size = size
		h.dirty = true
	}
	return 0
}

func (d *driveFS) Release(path string, fh uint64) int {
	d.mu.Lock()
	h := d.handles[fh]
	delete(d.handles, fh)
	delete(d.pending, pendingKey(path))
	d.mu.Unlock()
	if h == nil {
		return 0
	}
	if !h.dirty {
		_ = h.file.Close()
		_ = os.Remove(h.tempPath)
		return 0
	}
	if _, err := h.file.Seek(0, io.SeekStart); err != nil {
		_ = h.file.Close()
		// keep temp file for inspection
		return -fuse.EIO
	}
	if _, err := d.svc.SaveStreamFile(context.Background(), h.file, h.name, "", h.parentID, ""); err != nil {
		// Save failed: keep temp file in fuse-tmp/ for retry/manual recovery,
		// rename to <name>.<id>.unsaved so user can find it.
		_ = h.file.Close()
		failPath := h.tempPath + ".unsaved"
		_ = os.Rename(h.tempPath, failPath)
		return -fuse.EIO
	}
	_ = h.file.Close()
	_ = os.Remove(h.tempPath)
	return 0
}

func (d *driveFS) Unlink(path string) int {
	file, err := d.svc.ResolveFileByPath(context.Background(), path)
	if err != nil {
		return -fuse.ENOENT
	}
	if err := d.svc.TrashFile(context.Background(), file.ID); err != nil {
		return -fuse.EIO
	}
	return 0
}

func (d *driveFS) Mkdir(path string, mode uint32) int {
	parent, leaf := splitPath(path)
	if leaf == "" {
		return -fuse.EINVAL
	}
	parentID, err := d.svc.ResolveFolderByPath(context.Background(), parent)
	if err != nil {
		return -fuse.ENOENT
	}
	if _, err := d.svc.CreateFolder(context.Background(), drive.CreateFolderInput{ParentID: parentID, Name: leaf}); err != nil {
		return -fuse.EIO
	}
	return 0
}

func (d *driveFS) Rmdir(path string) int {
	folder, err := d.svc.ResolveFolderEntryByPath(context.Background(), path)
	if err != nil {
		return -fuse.ENOENT
	}
	if err := d.svc.TrashFolder(context.Background(), folder.ID); err != nil {
		return -fuse.EIO
	}
	return 0
}

func (d *driveFS) Rename(oldpath, newpath string) int {
	oldParent, oldLeaf := splitPath(oldpath)
	newParent, newLeaf := splitPath(newpath)
	if file, err := d.svc.ResolveFileByPath(context.Background(), oldpath); err == nil {
		if oldLeaf != newLeaf {
			if _, err := d.svc.RenameFile(context.Background(), drive.RenameInput{ID: file.ID, Name: newLeaf}); err != nil {
				return -fuse.EIO
			}
		}
		if oldParent != newParent {
			parentID, err := d.svc.ResolveFolderByPath(context.Background(), newParent)
			if err != nil {
				return -fuse.ENOENT
			}
			if _, err := d.svc.MoveFile(context.Background(), drive.MoveInput{ID: file.ID, NewParentID: parentID}); err != nil {
				return -fuse.EIO
			}
		}
		return 0
	}
	if folder, err := d.svc.ResolveFolderEntryByPath(context.Background(), oldpath); err == nil {
		if oldLeaf != newLeaf {
			if _, err := d.svc.RenameFolder(context.Background(), drive.RenameInput{ID: folder.ID, Name: newLeaf}); err != nil {
				return -fuse.EIO
			}
		}
		if oldParent != newParent {
			parentID, err := d.svc.ResolveFolderByPath(context.Background(), newParent)
			if err != nil {
				return -fuse.ENOENT
			}
			if _, err := d.svc.MoveFolder(context.Background(), drive.MoveInput{ID: folder.ID, NewParentID: parentID}); err != nil {
				return -fuse.EIO
			}
		}
		return 0
	}
	return -fuse.ENOENT
}

func randomTempName(name string) string {
	return name + ".tdtmp"
}

func (d *driveFS) Getattr(path string, stat *fuse.Stat_t, _ uint64) int {
	if path == "/" || strings.TrimPrefix(path, "/") == "" {
		stat.Mode = fuse.S_IFDIR | 0o777
		return 0
	}
	d.mu.Lock()
	if h, ok := d.pending[pendingKey(path)]; ok {
		d.mu.Unlock()
		stat.Mode = fuse.S_IFREG | 0o666
		stat.Size = h.size
		return 0
	}
	d.mu.Unlock()
	parent, leaf := splitPath(path)
	folderID, err := d.resolveFolder(context.Background(), parent)
	if err != nil {
		return -fuse.ENOENT
	}
	contents, err := d.svc.ListFolderContents(context.Background(), folderID)
	if err != nil {
		return -fuse.EIO
	}
	for _, folder := range contents.Folders {
		if folder.Name == leaf {
			stat.Mode = fuse.S_IFDIR | 0o777
			return 0
		}
	}
	for _, file := range contents.Files {
		if file.Name == leaf {
			stat.Mode = fuse.S_IFREG | 0o666
			stat.Size = file.Size
			return 0
		}
	}
	return -fuse.ENOENT
}

func (d *driveFS) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, _ int64, _ uint64) int {
	folderID, err := d.resolveFolder(context.Background(), path)
	if err != nil {
		return -fuse.ENOENT
	}
	contents, err := d.svc.ListFolderContents(context.Background(), folderID)
	if err != nil {
		return -fuse.EIO
	}
	fill(".", nil, 0)
	fill("..", nil, 0)
	for _, folder := range contents.Folders {
		stat := fuse.Stat_t{Mode: fuse.S_IFDIR | 0o777}
		if !fill(folder.Name, &stat, 0) {
			return 0
		}
	}
	for _, file := range contents.Files {
		stat := fuse.Stat_t{Mode: fuse.S_IFREG | 0o666, Size: file.Size}
		if !fill(file.Name, &stat, 0) {
			return 0
		}
	}
	d.mu.Lock()
	prefix := pendingKey(strings.TrimSuffix(path, "/")) + "/"
	for key, h := range d.pending {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		stat := fuse.Stat_t{Mode: fuse.S_IFREG | 0o666, Size: h.size}
		fill(h.name, &stat, 0)
	}
	d.mu.Unlock()
	return 0
}

func (d *driveFS) Read(path string, buff []byte, ofst int64, _ uint64) int {
	parent, leaf := splitPath(path)
	folderID, err := d.resolveFolder(context.Background(), parent)
	if err != nil {
		return -fuse.ENOENT
	}
	contents, err := d.svc.ListFolderContents(context.Background(), folderID)
	if err != nil {
		return -fuse.EIO
	}
	for _, file := range contents.Files {
		if file.Name != leaf {
			continue
		}
		buf := &chunkBuffer{}
		_, err := d.svc.StreamFromTelegram(context.Background(), file.ID, ofst, int64(len(buff)), buf)
		if err != nil {
			return -fuse.EIO
		}
		n := copy(buff, buf.bytes)
		return n
	}
	return -fuse.ENOENT
}

func (d *driveFS) resolveFolder(ctx context.Context, path string) (string, error) {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return "", nil
	}
	parts := strings.Split(clean, "/")
	current := ""
	for _, part := range parts {
		contents, err := d.svc.ListFolderContents(ctx, current)
		if err != nil {
			return "", err
		}
		match := ""
		for _, folder := range contents.Folders {
			if folder.Name == part {
				match = folder.ID
				break
			}
		}
		if match == "" {
			return "", errors.New("không tìm thấy thư mục")
		}
		current = match
	}
	return current, nil
}

type chunkBuffer struct {
	bytes []byte
}

func (c *chunkBuffer) Write(p []byte) (int, error) {
	c.bytes = append(c.bytes, p...)
	return len(p), nil
}

func splitPath(p string) (string, string) {
	clean := strings.TrimPrefix(p, "/")
	if clean == "" {
		return "/", ""
	}
	idx := strings.LastIndex(clean, "/")
	if idx < 0 {
		return "/", clean
	}
	return "/" + clean[:idx], clean[idx+1:]
}
