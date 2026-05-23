package telegram

import (
	"context"
	"errors"
	"io"

	"telegram-drive-agent/internal/storage"
)

var ErrNotImplemented = errors.New("telegram backend is not implemented yet")

// Backend is the future MTProto storage adapter. It will be implemented with gotd/td.
type Backend struct{}

func New() *Backend {
	return &Backend{}
}

func (b *Backend) PutObject(ctx context.Context, input storage.PutObjectInput) (storage.ObjectRef, error) {
	return storage.ObjectRef{}, ErrNotImplemented
}

func (b *Backend) GetObject(ctx context.Context, ref storage.ObjectRef) (io.ReadCloser, error) {
	return nil, ErrNotImplemented
}

func (b *Backend) GetObjectRange(ctx context.Context, ref storage.ObjectRef, start, end int64) (io.ReadCloser, error) {
	return nil, ErrNotImplemented
}

func (b *Backend) DeleteObject(ctx context.Context, ref storage.ObjectRef) error {
	return ErrNotImplemented
}

func (b *Backend) StatObject(ctx context.Context, ref storage.ObjectRef) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, ErrNotImplemented
}
