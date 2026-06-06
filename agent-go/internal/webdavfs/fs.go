package webdavfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/webdav"

	"telegram-drive-agent/internal/drive"
)

type FileSystem struct {
	svc *drive.Service
	mu  sync.Mutex
}

func New(svc *drive.Service) *FileSystem {
	return &FileSystem{svc: svc}
}

var errReadOnly = errors.New("WebDAV của Ổ Đĩa Cloud Ảo đang ở chế độ chỉ đọc")

func (f *FileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	parent, leaf := splitPath(name)
	if leaf == "" {
		return errors.New("tên thư mục trống")
	}
	parentID, err := f.resolveFolder(ctx, parent)
	if err != nil {
		return err
	}
	_, err = f.svc.CreateFolder(ctx, drive.CreateFolderInput{ParentID: parentID, Name: leaf})
	return err
}

func (f *FileSystem) RemoveAll(ctx context.Context, name string) error {
	cleaned := cleanPath(name)
	if cleaned == "" || cleaned == "/" {
		return errors.New("không thể xóa thư mục gốc")
	}
	parent, leaf := splitPath(cleaned)
	parentID, err := f.resolveFolder(ctx, parent)
	if err != nil {
		return err
	}
	contents, err := f.svc.ListFolderContents(ctx, parentID)
	if err != nil {
		return err
	}
	for _, folder := range contents.Folders {
		if folder.Name == leaf {
			return f.svc.TrashFolder(ctx, folder.ID)
		}
	}
	for _, file := range contents.Files {
		if file.Name == leaf {
			return f.svc.TrashFile(ctx, file.ID)
		}
	}
	return os.ErrNotExist
}

func (f *FileSystem) Rename(ctx context.Context, oldName, newName string) error {
	oldParent, oldLeaf := splitPath(oldName)
	newParent, newLeaf := splitPath(newName)
	parentID, err := f.resolveFolder(ctx, oldParent)
	if err != nil {
		return err
	}
	contents, err := f.svc.ListFolderContents(ctx, parentID)
	if err != nil {
		return err
	}
	for _, folder := range contents.Folders {
		if folder.Name != oldLeaf {
			continue
		}
		if oldLeaf != newLeaf {
			if _, err := f.svc.RenameFolder(ctx, drive.RenameInput{ID: folder.ID, Name: newLeaf}); err != nil {
				return err
			}
		}
		if newParent != oldParent {
			newParentID, err := f.resolveFolder(ctx, newParent)
			if err != nil {
				return err
			}
			if _, err := f.svc.MoveFolder(ctx, drive.MoveInput{ID: folder.ID, NewParentID: newParentID}); err != nil {
				return err
			}
		}
		return nil
	}
	for _, file := range contents.Files {
		if file.Name != oldLeaf {
			continue
		}
		if oldLeaf != newLeaf {
			if _, err := f.svc.RenameFile(ctx, drive.RenameInput{ID: file.ID, Name: newLeaf}); err != nil {
				return err
			}
		}
		if newParent != oldParent {
			newParentID, err := f.resolveFolder(ctx, newParent)
			if err != nil {
				return err
			}
			if _, err := f.svc.MoveFile(ctx, drive.MoveInput{ID: file.ID, NewParentID: newParentID}); err != nil {
				return err
			}
		}
		return nil
	}
	return os.ErrNotExist
}

func (f *FileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	cleaned := cleanPath(name)
	if cleaned == "" || cleaned == "/" {
		return rootInfo(), nil
	}
	parent, leaf := splitPath(cleaned)
	parentID, err := f.resolveFolder(ctx, parent)
	if err != nil {
		return nil, err
	}
	contents, err := f.svc.ListFolderContents(ctx, parentID)
	if err != nil {
		return nil, err
	}
	for _, folder := range contents.Folders {
		if folder.Name == leaf {
			return folderInfo(folder), nil
		}
	}
	for _, file := range contents.Files {
		if file.Name == leaf {
			return fileInfo(file), nil
		}
	}
	return nil, os.ErrNotExist
}

func (f *FileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	cleaned := cleanPath(name)
	wantWrite := flag&(os.O_WRONLY|os.O_RDWR) != 0
	wantCreate := flag&os.O_CREATE != 0
	if cleaned == "" || cleaned == "/" {
		if wantWrite {
			return nil, errors.New("không thể ghi vào thư mục gốc")
		}
		return f.openFolder(ctx, "", "/")
	}
	parent, leaf := splitPath(cleaned)
	parentID, err := f.resolveFolder(ctx, parent)
	if err != nil {
		if wantCreate && parent != "/" {
			return nil, err
		}
		if !wantCreate {
			return nil, err
		}
	}
	contents, err := f.svc.ListFolderContents(ctx, parentID)
	if err != nil {
		return nil, err
	}
	for _, folder := range contents.Folders {
		if folder.Name == leaf {
			if wantWrite {
				return nil, errors.New("không thể ghi đè thư mục")
			}
			return f.openFolder(ctx, folder.ID, cleaned)
		}
	}
	for _, file := range contents.Files {
		if file.Name == leaf {
			if wantWrite {
				return f.newWriteFile(ctx, parentID, leaf), nil
			}
			return f.openFile(ctx, file)
		}
	}
	if wantCreate {
		return f.newWriteFile(ctx, parentID, leaf), nil
	}
	return nil, os.ErrNotExist
}

func (f *FileSystem) newWriteFile(ctx context.Context, parentID, name string) webdav.File {
	tempPath := filepath.Join(os.TempDir(), "td-webdav-"+strings.ReplaceAll(name, string(filepath.Separator), "_")+"-"+randomTempSuffix())
	tempFile, err := os.CreateTemp(filepath.Dir(tempPath), filepath.Base(tempPath)+"-")
	if err != nil {
		// Fallback: keep buffer-in-RAM behavior so Close still works for tiny files.
		return &writeFile{ctx: detachContext(ctx), svc: f.svc, parentID: parentID, name: name, info: &entryInfo{name: name, mode: 0o644, modTime: time.Now()}}
	}
	return &writeFile{ctx: detachContext(ctx), svc: f.svc, parentID: parentID, name: name, info: &entryInfo{name: name, mode: 0o644, modTime: time.Now()}, file: tempFile, path: tempFile.Name()}
}

func detachContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func randomTempSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (f *FileSystem) openFolder(ctx context.Context, folderID, name string) (webdav.File, error) {
	contents, err := f.svc.ListFolderContents(ctx, folderID)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, 0, len(contents.Folders)+len(contents.Files))
	for _, folder := range contents.Folders {
		infos = append(infos, folderInfo(folder))
	}
	for _, file := range contents.Files {
		infos = append(infos, fileInfo(file))
	}
	return &dir{name: name, entries: infos}, nil
}

func (f *FileSystem) openFile(ctx context.Context, file drive.File) (webdav.File, error) {
	return &remoteFile{ctx: ctx, svc: f.svc, file: file, info: fileInfo(file)}, nil
}

func (f *FileSystem) resolveFolder(ctx context.Context, name string) (string, error) {
	cleaned := cleanPath(name)
	if cleaned == "" || cleaned == "/" {
		return "", nil
	}
	parts := strings.Split(strings.Trim(cleaned, "/"), "/")
	currentID := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		contents, err := f.svc.ListFolderContents(ctx, currentID)
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
			return "", os.ErrNotExist
		}
		currentID = match
	}
	return currentID, nil
}

type dir struct {
	name    string
	entries []os.FileInfo
	pos     int
}

func (d *dir) Close() error                { return nil }
func (d *dir) Read(p []byte) (int, error)  { return 0, io.EOF }
func (d *dir) Write(p []byte) (int, error) { return 0, errReadOnly }
func (d *dir) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("không thể seek thư mục")
}
func (d *dir) Stat() (os.FileInfo, error) { return rootInfo(), nil }
func (d *dir) Readdir(count int) ([]os.FileInfo, error) {
	if d.pos >= len(d.entries) {
		if count <= 0 {
			return []os.FileInfo{}, nil
		}
		return nil, io.EOF
	}
	end := len(d.entries)
	if count > 0 && d.pos+count < end {
		end = d.pos + count
	}
	chunk := d.entries[d.pos:end]
	d.pos = end
	return chunk, nil
}

type remoteFile struct {
	ctx    context.Context
	svc    *drive.Service
	file   drive.File
	info   os.FileInfo
	offset int64
}

func (r *remoteFile) Close() error                { return nil }
func (r *remoteFile) Write(p []byte) (int, error) { return 0, errReadOnly }
func (r *remoteFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("không phải thư mục")
}
func (r *remoteFile) Stat() (os.FileInfo, error) { return r.info, nil }

func (r *remoteFile) Seek(offset int64, whence int) (int64, error) {
	size := r.file.Size
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = size + offset
	}
	if r.offset < 0 {
		r.offset = 0
	}
	if r.offset > size {
		r.offset = size
	}
	return r.offset, nil
}

func (r *remoteFile) Read(p []byte) (int, error) {
	if r.offset >= r.file.Size {
		return 0, io.EOF
	}
	stream, err := r.svc.GetStreamableFile(r.ctx, r.file.ID)
	if err != nil {
		return 0, err
	}
	if stream.Source == drive.StreamFromCache {
		f, err := os.Open(stream.LocalPath)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		if _, err := f.Seek(r.offset, io.SeekStart); err != nil {
			return 0, err
		}
		n, err := f.Read(p)
		r.offset += int64(n)
		return n, err
	}
	buf := &bytesBuffer{}
	want := int64(len(p))
	if r.offset+want > r.file.Size {
		want = r.file.Size - r.offset
	}
	if _, err := r.svc.StreamFromTelegram(r.ctx, r.file.ID, r.offset, want, buf); err != nil {
		return 0, err
	}
	n := copy(p, buf.data)
	r.offset += int64(n)
	return n, nil
}

type writeFile struct {
	ctx      context.Context
	svc      *drive.Service
	parentID string
	name     string
	buf      []byte
	closed   bool
	info     *entryInfo
	file     *os.File
	path     string
	written  int64
}

func (w *writeFile) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			w.file.Close()
			os.Remove(w.path)
			return err
		}
		if _, err := w.file.Seek(0, io.SeekStart); err != nil {
			w.file.Close()
			os.Remove(w.path)
			return err
		}
		_, err := w.svc.SaveStreamFile(w.ctx, w.file, w.name, "", w.parentID, "")
		closeErr := w.file.Close()
		removeErr := os.Remove(w.path)
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		_ = removeErr
		return nil
	}
	reader := &bytesReader{data: w.buf}
	_, err := w.svc.SaveStreamFile(w.ctx, reader, w.name, "", w.parentID, "")
	return err
}

func (w *writeFile) Read(p []byte) (int, error) { return 0, io.EOF }
func (w *writeFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("không phải thư mục")
}
func (w *writeFile) Seek(offset int64, whence int) (int64, error) {
	if w.file != nil {
		return w.file.Seek(offset, whence)
	}
	return int64(len(w.buf)), nil
}
func (w *writeFile) Stat() (os.FileInfo, error) { return w.info, nil }

func (w *writeFile) Write(p []byte) (int, error) {
	if w.file != nil {
		n, err := w.file.Write(p)
		w.written += int64(n)
		w.info.size = w.written
		return n, err
	}
	w.buf = append(w.buf, p...)
	w.info.size = int64(len(w.buf))
	return len(p), nil
}

type bytesReader struct {
	data []byte
	off  int
}

func (b *bytesReader) Read(p []byte) (int, error) {
	if b.off >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.off:])
	b.off += n
	return n, nil
}

type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

type entryInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (e *entryInfo) Name() string       { return e.name }
func (e *entryInfo) Size() int64        { return e.size }
func (e *entryInfo) Mode() os.FileMode  { return e.mode }
func (e *entryInfo) ModTime() time.Time { return e.modTime }
func (e *entryInfo) IsDir() bool        { return e.isDir }
func (e *entryInfo) Sys() interface{}   { return nil }

func rootInfo() os.FileInfo {
	return &entryInfo{name: "/", mode: os.ModeDir | 0o755, isDir: true, modTime: time.Now()}
}

func folderInfo(folder drive.Folder) os.FileInfo {
	return &entryInfo{name: folder.Name, mode: os.ModeDir | 0o755, isDir: true, modTime: time.Unix(folder.UpdatedAt, 0)}
}

func fileInfo(file drive.File) os.FileInfo {
	return &entryInfo{name: file.Name, size: file.Size, mode: 0o644, modTime: time.Unix(file.UpdatedAt, 0)}
}

func cleanPath(name string) string {
	return path.Clean("/" + strings.TrimPrefix(name, "/"))
}

func splitPath(name string) (string, string) {
	cleaned := cleanPath(name)
	if cleaned == "/" {
		return "/", ""
	}
	dir, leaf := path.Split(cleaned)
	return strings.TrimSuffix(dir, "/"), leaf
}

func describeError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
