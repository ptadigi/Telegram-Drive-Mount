package drive

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type StorageSettings struct {
	PeerKind   string `json:"peer_kind"`
	ChannelID  int64  `json:"channel_id"`
	AccessHash int64  `json:"access_hash"`
	Title      string `json:"title,omitempty"`
	UpdatedAt  int64  `json:"updated_at"`
}

type UpdateStorageInput struct {
	PeerKind   string `json:"peer_kind"`
	ChannelID  int64  `json:"channel_id"`
	AccessHash int64  `json:"access_hash"`
	Title      string `json:"title"`
}

func (s *Service) GetStorageSettings(ctx context.Context) (StorageSettings, error) {
	row := s.db.QueryRowContext(ctx, `SELECT peer_kind, channel_id, access_hash, COALESCE(title, ''), updated_at FROM storage_settings WHERE id = 'default'`)
	var settings StorageSettings
	if err := row.Scan(&settings.PeerKind, &settings.ChannelID, &settings.AccessHash, &settings.Title, &settings.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StorageSettings{PeerKind: "self"}, nil
		}
		return StorageSettings{}, err
	}
	return settings, nil
}

func (s *Service) UpdateStorageSettings(ctx context.Context, input UpdateStorageInput) (StorageSettings, error) {
	kind := strings.ToLower(strings.TrimSpace(input.PeerKind))
	if kind != "self" && kind != "channel" {
		return StorageSettings{}, fmt.Errorf("kiểu lưu trữ không hợp lệ")
	}
	if kind == "channel" && input.ChannelID == 0 {
		return StorageSettings{}, fmt.Errorf("thiếu channel_id")
	}
	now := time.Now().Unix()
	settings := StorageSettings{PeerKind: kind, ChannelID: input.ChannelID, AccessHash: input.AccessHash, Title: strings.TrimSpace(input.Title), UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO storage_settings (id, peer_kind, channel_id, access_hash, title, updated_at)
		VALUES ('default', ?, ?, ?, NULLIF(?, ''), ?)
		ON CONFLICT(id) DO UPDATE SET peer_kind = excluded.peer_kind, channel_id = excluded.channel_id, access_hash = excluded.access_hash, title = excluded.title, updated_at = excluded.updated_at
	`, settings.PeerKind, settings.ChannelID, settings.AccessHash, settings.Title, now)
	if err != nil {
		return StorageSettings{}, fmt.Errorf("ghi cấu hình lưu trữ: %w", err)
	}
	s.events.Publish("storage.updated", settings)
	return settings, nil
}
