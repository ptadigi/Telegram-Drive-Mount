package telegramstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
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

type StreamResult = drive.StreamResult

func (s *Service) StreamFromSavedMessages(ctx context.Context, messageID int, offset int64, length int64, w io.Writer) (drive.StreamResult, error) {
	if s.cfg.Telegram.APIID == 0 || s.cfg.Telegram.APIHash == "" {
		return drive.StreamResult{}, errors.New("chưa cấu hình API Telegram cho Go Agent")
	}
	if messageID <= 0 {
		return drive.StreamResult{}, errors.New("thiếu Telegram message id")
	}
	client := telegram.NewClient(s.cfg.Telegram.APIID, s.cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: s.cfg.Telegram.SessionPath},
	})
	var result StreamResult
	err := client.Run(ctx, func(runCtx context.Context) error {
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
		var doc *tg.Document
		switch m := messages.(type) {
		case *tg.MessagesMessages:
			doc = documentFromMessages(m.Messages, messageID)
		case *tg.MessagesMessagesSlice:
			doc = documentFromMessages(m.Messages, messageID)
		default:
			return fmt.Errorf("loại trả về Telegram không hỗ trợ: %T", messages)
		}
		if doc == nil {
			return errors.New("không tìm thấy file Telegram tương ứng")
		}
		result.Size = doc.Size
		result.MimeType = doc.MimeType
		location := &tg.InputDocumentFileLocation{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}
		return streamDocument(runCtx, api, location, doc.Size, offset, length, w)
	})
	return result, err
}

func streamDocument(ctx context.Context, api *tg.Client, location tg.InputFileLocationClass, totalSize, offset, length int64, w io.Writer) error {
	const chunkLimit = 1024 * 1024
	const chunkAlign = 1024 * 1024
	if offset < 0 {
		offset = 0
	}
	if length <= 0 {
		length = totalSize - offset
	}
	end := offset + length
	if totalSize > 0 && end > totalSize {
		end = totalSize
	}
	current := (offset / chunkAlign) * chunkAlign
	skip := offset - current
	for current < end {
		req := &tg.UploadGetFileRequest{Location: location, Offset: current, Limit: chunkLimit, Precise: true}
		resp, err := api.UploadGetFile(ctx, req)
		if err != nil {
			return fmt.Errorf("đọc chunk Telegram: %w", err)
		}
		fileResp, ok := resp.(*tg.UploadFile)
		if !ok {
			return fmt.Errorf("Telegram trả về loại không hỗ trợ stream: %T", resp)
		}
		bytes := fileResp.Bytes
		if len(bytes) == 0 {
			break
		}
		from := int64(0)
		if skip > 0 {
			if skip >= int64(len(bytes)) {
				skip -= int64(len(bytes))
				current += int64(len(bytes))
				continue
			}
			from = skip
			skip = 0
		}
		remaining := end - (current + from)
		if remaining <= 0 {
			break
		}
		take := int64(len(bytes)) - from
		if take > remaining {
			take = remaining
		}
		if _, err := w.Write(bytes[from : from+take]); err != nil {
			return err
		}
		current += int64(len(bytes))
	}
	return nil
}

func documentFromMessages(messages []tg.MessageClass, messageID int) *tg.Document {
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
		return doc
	}
	return nil
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
