package telegramstorage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/message/unpack"

	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/drive"
)

var ErrUnauthorized = errors.New("Telegram chưa được kết nối hoặc session đã hết hạn")

type Service struct {
	cfg config.Config
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) UploadToSavedMessages(ctx context.Context, localPath string, originalName string) (drive.UploadedObject, error) {
	if s.cfg.Telegram.APIID == 0 || s.cfg.Telegram.APIHash == "" {
		return drive.UploadedObject{}, errors.New("chưa cấu hình API Telegram cho Go Agent")
	}

	client := telegram.NewClient(s.cfg.Telegram.APIID, s.cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: s.cfg.Telegram.SessionPath},
	})

	var uploaded drive.UploadedObject
	err := client.Run(ctx, func(runCtx context.Context) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}

		caption := fmt.Sprintf("TD_OBJECT:%s", filepath.Base(localPath))
		msg, err := unpack.Message(message.NewSender(client.API()).Self().Upload(message.FromPath(localPath)).File(runCtx, styling.Plain(caption)))
		if err != nil {
			return fmt.Errorf("upload file lên Telegram Saved Messages: %w", err)
		}

		uploaded.MessageID = msg.ID
		uploaded.FileID = fmt.Sprintf("saved:%d", msg.ID)
		return nil
	})
	if err != nil {
		return drive.UploadedObject{}, err
	}

	return uploaded, nil
}
