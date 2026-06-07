package vfs

import (
	"context"
	"io"

	"telegram-drive-agent/internal/drive"
)

// Backend abstracts the read/write operations the FUSE filesystem needs.
// The default implementation in this package wraps drive.Service for the
// single-process case (VPS hosting both metadata and FUSE). A future
// remote implementation will be added so td-agent --remote can call a
// VPS over HTTPS instead of touching the local DB.
type Backend interface {
	ListFolderContents(ctx context.Context, folderID string) (drive.FolderContents, error)
	ResolveFolderByPath(ctx context.Context, p string) (string, error)
	ResolveFileByPath(ctx context.Context, p string) (drive.File, error)
	ResolveFolderEntryByPath(ctx context.Context, p string) (drive.Folder, error)
	StreamFromTelegram(ctx context.Context, fileID string, offset, length int64, w io.Writer) (drive.StreamResult, error)
	SaveStreamFile(ctx context.Context, data io.Reader, filename, mimeHint, folderID, relativePath string) (drive.File, error)
	CreateFolder(ctx context.Context, input drive.CreateFolderInput) (drive.Folder, error)
	TrashFile(ctx context.Context, id string) error
	TrashFolder(ctx context.Context, id string) error
	RenameFile(ctx context.Context, input drive.RenameInput) (drive.File, error)
	RenameFolder(ctx context.Context, input drive.RenameInput) (drive.Folder, error)
	MoveFile(ctx context.Context, input drive.MoveInput) (drive.File, error)
	MoveFolder(ctx context.Context, input drive.MoveInput) (drive.Folder, error)
}

// localBackend wraps the in-process drive.Service. It exists so the FUSE
// layer is decoupled from drive.Service directly; this is what allows the
// remote backend to drop in later without touching driveFS internals.
type localBackend struct {
	svc *drive.Service
}

func newLocalBackend(svc *drive.Service) *localBackend { return &localBackend{svc: svc} }

func (b *localBackend) ListFolderContents(ctx context.Context, folderID string) (drive.FolderContents, error) {
	return b.svc.ListFolderContents(ctx, folderID)
}
func (b *localBackend) ResolveFolderByPath(ctx context.Context, p string) (string, error) {
	return b.svc.ResolveFolderByPath(ctx, p)
}
func (b *localBackend) ResolveFileByPath(ctx context.Context, p string) (drive.File, error) {
	return b.svc.ResolveFileByPath(ctx, p)
}
func (b *localBackend) ResolveFolderEntryByPath(ctx context.Context, p string) (drive.Folder, error) {
	return b.svc.ResolveFolderEntryByPath(ctx, p)
}
func (b *localBackend) StreamFromTelegram(ctx context.Context, fileID string, offset, length int64, w io.Writer) (drive.StreamResult, error) {
	return b.svc.StreamFromTelegram(ctx, fileID, offset, length, w)
}
func (b *localBackend) SaveStreamFile(ctx context.Context, data io.Reader, filename, mimeHint, folderID, relativePath string) (drive.File, error) {
	return b.svc.SaveStreamFile(ctx, data, filename, mimeHint, folderID, relativePath)
}
func (b *localBackend) CreateFolder(ctx context.Context, input drive.CreateFolderInput) (drive.Folder, error) {
	return b.svc.CreateFolder(ctx, input)
}
func (b *localBackend) TrashFile(ctx context.Context, id string) error  { return b.svc.TrashFile(ctx, id) }
func (b *localBackend) TrashFolder(ctx context.Context, id string) error { return b.svc.TrashFolder(ctx, id) }
func (b *localBackend) RenameFile(ctx context.Context, input drive.RenameInput) (drive.File, error) {
	return b.svc.RenameFile(ctx, input)
}
func (b *localBackend) RenameFolder(ctx context.Context, input drive.RenameInput) (drive.Folder, error) {
	return b.svc.RenameFolder(ctx, input)
}
func (b *localBackend) MoveFile(ctx context.Context, input drive.MoveInput) (drive.File, error) {
	return b.svc.MoveFile(ctx, input)
}
func (b *localBackend) MoveFolder(ctx context.Context, input drive.MoveInput) (drive.Folder, error) {
	return b.svc.MoveFolder(ctx, input)
}