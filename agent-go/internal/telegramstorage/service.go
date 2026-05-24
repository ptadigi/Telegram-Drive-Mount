package telegramstorage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/message/unpack"
	"github.com/gotd/td/tg"

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

func (s *Service) DownloadFromSavedMessages(ctx context.Context, messageID int, targetPath string) error {
	if s.cfg.Telegram.APIID == 0 || s.cfg.Telegram.APIHash == "" {
		return errors.New("chưa cấu hình API Telegram cho Go Agent")
	}
	if messageID <= 0 {
		return errors.New("thiếu Telegram message id")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("tạo thư mục cache: %w", err)
	}
	client := telegram.NewClient(s.cfg.Telegram.APIID, s.cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: s.cfg.Telegram.SessionPath},
	})
	return client.Run(ctx, func(runCtx context.Context) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		api := client.API()
		messages, err := api.MessagesGetMessages(runCtx, []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}})
		if err != nil {
			return fmt.Errorf("đọc tin nhắn Telegram: %w", err)
		}
		var location tg.InputFileLocationClass
		switch m := messages.(type) {
		case *tg.MessagesMessages:
			location = locationFromMessages(m.Messages, messageID)
		case *tg.MessagesMessagesSlice:
			location = locationFromMessages(m.Messages, messageID)
		default:
			return fmt.Errorf("loại trả về Telegram không hỗ trợ: %T", messages)
		}
		if location == nil {
			return errors.New("không tìm thấy file Telegram tương ứng")
		}
		_, err = downloader.NewDownloader().Download(api, location).ToPath(runCtx, targetPath)
		if err != nil {
			return fmt.Errorf("tải file Telegram: %w", err)
		}
		return nil
	})
}

func locationFromMessages(messages []tg.MessageClass, messageID int) tg.InputFileLocationClass {
	for _, msg := range messages {
		concrete, ok := msg.(*tg.Message)
		if !ok || concrete.ID != messageID {
			continue
		}
		media, ok := concrete.Media.(*tg.MessageMediaDocument)
		if !ok {
			continue
		}
		doc, ok := media.Document.AsNotEmpty()
		if !ok {
			continue
		}
		return &tg.InputDocumentFileLocation{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}
	}
	return nil
}
