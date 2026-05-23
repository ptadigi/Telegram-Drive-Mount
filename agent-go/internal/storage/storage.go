package storage

import (
	"context"
	"io"
	"time"
)

// ObjectRef identifies one physical object in a storage backend.
type ObjectRef struct {
	Backend string `json:"backend"`
	Bucket  string `json:"bucket,omitempty"`
	Key     string `json:"key"`
}

type ObjectInfo struct {
	Ref         ObjectRef `json:"ref"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PutObjectInput struct {
	Name        string
	Size        int64
	ContentType string
	Reader      io.Reader
}

// Backend abstracts Telegram as object storage so future backends can be added.
type Backend interface {
	PutObject(ctx context.Context, input PutObjectInput) (ObjectRef, error)
	GetObject(ctx context.Context, ref ObjectRef) (io.ReadCloser, error)
	GetObjectRange(ctx context.Context, ref ObjectRef, start, end int64) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, ref ObjectRef) error
	StatObject(ctx context.Context, ref ObjectRef) (ObjectInfo, error)
}
