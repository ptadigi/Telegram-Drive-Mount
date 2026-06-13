package telegramstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/unpack"
	"github.com/gotd/td/tg"

	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/drive"
	"telegram-drive-agent/internal/secret"
)

var ErrUnauthorized = errors.New("Telegram chưa được kết nối hoặc session đã hết hạn")

type Service struct {
	cfg config.Config

	// clientMu serializes every Telegram client.Run lifecycle inside this
	// process so two goroutines never race on the on-disk session file
	// or fight over reconnects. Long-running ops still complete; new ops
	// just queue behind them.
	clientMu sync.Mutex
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

// runClient serializes Telegram calls and creates the gotd client with the
// session storage path. The session file is shared, so concurrent runs would
// corrupt it; the mutex makes the call site explicit and avoids surprises.
func (s *Service) runClient(ctx context.Context, fn func(runCtx context.Context, client *telegram.Client) error) error {
	if s.cfg.Telegram.APIID == 0 || s.cfg.Telegram.APIHash == "" {
		return errors.New("chưa cấu hình API Telegram cho Go Agent")
	}
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	storage, err := newSessionStorage(s.cfg.Telegram.SessionPath)
	if err != nil {
		return err
	}
	client := telegram.NewClient(s.cfg.Telegram.APIID, s.cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: storage,
	})
	return client.Run(ctx, func(runCtx context.Context) error {
		return fn(runCtx, client)
	})
}

func newSessionStorage(path string) (session.Storage, error) {
	key, err := secret.LoadKey()
	if err != nil {
		return nil, err
	}
	if key == nil {
		// No env key configured: fall back to gotd's plain file storage so
		// existing dev setups keep working. README warns the user to set
		// TD_AGENT_SESSION_KEY in production.
		return &telegram.FileSessionStorage{Path: path}, nil
	}
	return &secret.EncryptedSessionStorage{Path: path, Key: key}, nil
}

func (s *Service) UploadToSavedMessages(ctx context.Context, localPath string, originalName string) (drive.UploadedObject, error) {
	var uploaded drive.UploadedObject
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		inputFile, err := message.NewSender(client.API()).Self().Upload(message.FromPath(localPath)).AsInputFile(runCtx)
		if err != nil {
			return fmt.Errorf("upload file lên Telegram Saved Messages: %w", err)
		}
		msg, err := unpack.Message(message.NewSender(client.API()).Self().Media(runCtx, message.File(inputFile).Filename(telegramFilename(originalName))))
		if err != nil {
			return fmt.Errorf("gửi file lên Telegram Saved Messages: %w", err)
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

func telegramFilename(originalName string) string {
	name := filepath.Base(originalName)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "telegram-drive-file"
	}
	return name
}

func (s *Service) DownloadFromSavedMessages(ctx context.Context, messageID int, targetPath string) error {
	if messageID <= 0 {
		return errors.New("thiếu Telegram message id")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("tạo thư mục cache: %w", err)
	}
	return s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
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
	if messageID <= 0 {
		return drive.StreamResult{}, errors.New("thiếu Telegram message id")
	}
	var result StreamResult
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
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

func (s *Service) UploadToPeer(ctx context.Context, peer drive.StoragePeer, localPath string, originalName string) (drive.UploadedObject, error) {
	if peer.Kind != "channel" || peer.ChannelID == 0 {
		return s.UploadToSavedMessages(ctx, localPath, originalName)
	}
	var uploaded drive.UploadedObject
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		channel := &tg.InputPeerChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash}
		inputFile, err := message.NewSender(client.API()).To(channel).Upload(message.FromPath(localPath)).AsInputFile(runCtx)
		if err != nil {
			return fmt.Errorf("upload file lên channel Telegram: %w", err)
		}
		msg, err := unpack.Message(message.NewSender(client.API()).To(channel).Media(runCtx, message.File(inputFile).Filename(telegramFilename(originalName))))
		if err != nil {
			return fmt.Errorf("gửi file lên channel Telegram: %w", err)
		}
		uploaded.MessageID = msg.ID
		uploaded.FileID = fmt.Sprintf("channel:%d:%d", peer.ChannelID, msg.ID)
		uploaded.ChannelID = peer.ChannelID
		uploaded.AccessHash = peer.AccessHash
		return nil
	})
	if err != nil {
		return drive.UploadedObject{}, err
	}
	return uploaded, nil
}

func (s *Service) DownloadFromPeer(ctx context.Context, peer drive.StoragePeer, messageID int, targetPath string) error {
	if peer.Kind != "channel" || peer.ChannelID == 0 {
		return s.DownloadFromSavedMessages(ctx, messageID, targetPath)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("tạo thư mục cache: %w", err)
	}
	return s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		api := client.API()
		channel := &tg.InputChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash}
		messages, err := api.ChannelsGetMessages(runCtx, &tg.ChannelsGetMessagesRequest{Channel: channel, ID: []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}}})
		if err != nil {
			return fmt.Errorf("đọc tin nhắn channel: %w", err)
		}
		location := locationFromChannelMessages(messages, messageID)
		if location == nil {
			return errors.New("không tìm thấy file trên channel")
		}
		_, err = downloader.NewDownloader().Download(api, location).ToPath(runCtx, targetPath)
		if err != nil {
			return fmt.Errorf("tải file channel: %w", err)
		}
		return nil
	})
}

func (s *Service) StreamFromPeer(ctx context.Context, peer drive.StoragePeer, messageID int, offset int64, length int64, w io.Writer) (drive.StreamResult, error) {
	if peer.Kind != "channel" || peer.ChannelID == 0 {
		return s.StreamFromSavedMessages(ctx, messageID, offset, length, w)
	}
	var result drive.StreamResult
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		api := client.API()
		channel := &tg.InputChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash}
		messages, err := api.ChannelsGetMessages(runCtx, &tg.ChannelsGetMessagesRequest{Channel: channel, ID: []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}}})
		if err != nil {
			return fmt.Errorf("đọc tin nhắn channel: %w", err)
		}
		doc := documentFromChannelMessages(messages, messageID)
		if doc == nil {
			return errors.New("không tìm thấy file trên channel")
		}
		result.Size = doc.Size
		result.MimeType = doc.MimeType
		location := &tg.InputDocumentFileLocation{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}
		return streamDocument(runCtx, api, location, doc.Size, offset, length, w)
	})
	return result, err
}

func locationFromChannelMessages(resp tg.MessagesMessagesClass, messageID int) tg.InputFileLocationClass {
	switch m := resp.(type) {
	case *tg.MessagesMessages:
		return locationFromMessages(m.Messages, messageID)
	case *tg.MessagesMessagesSlice:
		return locationFromMessages(m.Messages, messageID)
	case *tg.MessagesChannelMessages:
		return locationFromMessages(m.Messages, messageID)
	}
	return nil
}

func documentFromChannelMessages(resp tg.MessagesMessagesClass, messageID int) *tg.Document {
	switch m := resp.(type) {
	case *tg.MessagesMessages:
		return documentFromMessages(m.Messages, messageID)
	case *tg.MessagesMessagesSlice:
		return documentFromMessages(m.Messages, messageID)
	case *tg.MessagesChannelMessages:
		return documentFromMessages(m.Messages, messageID)
	}
	return nil
}

type CreatedChannel = drive.CreatedChannel

func (s *Service) CreateStorageChannel(ctx context.Context, title string) (drive.CreatedChannel, error) {
	if title == "" {
		title = "Ổ Đĩa Cloud Ảo"
	}
	var created drive.CreatedChannel
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		api := client.API()
		updates, err := api.ChannelsCreateChannel(runCtx, &tg.ChannelsCreateChannelRequest{
			Title:     title,
			About:     "Ổ Đĩa Cloud Ảo private storage",
			Megagroup: false,
			Broadcast: true,
		})
		if err != nil {
			return fmt.Errorf("tạo channel Telegram: %w", err)
		}
		channel := findChannelFromUpdates(updates)
		if channel == nil {
			return errors.New("không lấy được channel vừa tạo")
		}
		created = drive.CreatedChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash, Title: channel.Title}
		return nil
	})
	if err != nil {
		return drive.CreatedChannel{}, err
	}
	return created, nil
}

// VerifyChannel calls Telegram to ensure the given channel id+access_hash actually resolves.
// This filters out cases where a freshly-created channel is not yet visible to MTProto.
func (s *Service) VerifyChannel(ctx context.Context, channelID int64, accessHash int64) error {
	if channelID == 0 {
		return errors.New("channel_id rỗng")
	}
	return s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		api := client.API()
		_, err := api.ChannelsGetFullChannel(runCtx, &tg.InputChannel{ChannelID: channelID, AccessHash: accessHash})
		if err != nil {
			return fmt.Errorf("xác minh channel: %w", err)
		}
		return nil
	})
}

func findChannelFromUpdates(value tg.UpdatesClass) *tg.Channel {
	updates, ok := value.(*tg.Updates)
	if !ok {
		combined, ok2 := value.(*tg.UpdatesCombined)
		if !ok2 {
			return nil
		}
		for _, chat := range combined.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				return ch
			}
		}
		return nil
	}
	for _, chat := range updates.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			return ch
		}
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

// ScanChannelHistory reads the storage channel history and returns media
// messages whose ID is greater than afterMessageID (0 = from the beginning),
// oldest-first, up to limit. Used to import files uploaded directly from a
// native Telegram client. Read-only; respects FLOOD_WAIT via the gotd client.
func (s *Service) ScanChannelHistory(ctx context.Context, peer drive.StoragePeer, afterMessageID int, limit int) ([]drive.ChannelFile, error) {
	if peer.Kind != "channel" || peer.ChannelID == 0 {
		return nil, nil // only channel storage supports scanning
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var out []drive.ChannelFile
	err := s.runClient(ctx, func(runCtx context.Context, client *telegram.Client) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return fmt.Errorf("kiểm tra session Telegram: %w", err)
		}
		if !status.Authorized {
			return ErrUnauthorized
		}
		api := client.API()
		inputPeer := &tg.InputPeerChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash}
		// MinID returns messages with ID strictly greater than afterMessageID.
		// AddOffset=-limit + OffsetID make Telegram return the *oldest* unseen
		// page first, so we import in chronological order.
		resp, err := api.MessagesGetHistory(runCtx, &tg.MessagesGetHistoryRequest{
			Peer:      inputPeer,
			OffsetID:  afterMessageID + 1,
			AddOffset: -limit,
			Limit:     limit,
			MinID:     afterMessageID,
		})
		if err != nil {
			return fmt.Errorf("đọc lịch sử channel: %w", err)
		}
		out = channelFilesFromHistory(resp, afterMessageID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func channelFilesFromHistory(resp tg.MessagesMessagesClass, afterMessageID int) []drive.ChannelFile {
	var msgs []tg.MessageClass
	switch m := resp.(type) {
	case *tg.MessagesMessages:
		msgs = m.Messages
	case *tg.MessagesMessagesSlice:
		msgs = m.Messages
	case *tg.MessagesChannelMessages:
		msgs = m.Messages
	default:
		return nil
	}
	var files []drive.ChannelFile
	for _, raw := range msgs {
		msg, ok := raw.(*tg.Message)
		if !ok || msg.ID <= afterMessageID {
			continue
		}
		cf, ok := channelFileFromMessage(msg)
		if !ok {
			continue
		}
		files = append(files, cf)
	}
	// Telegram returns newest-first; import oldest-first for stable ordering.
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	return files
}

// channelFileFromMessage extracts a downloadable media file (document, photo,
// video, audio…) from a channel message. Documents carry a filename; photos
// and other media get a generated name.
func channelFileFromMessage(msg *tg.Message) (drive.ChannelFile, bool) {
	switch media := msg.Media.(type) {
	case *tg.MessageMediaDocument:
		doc, ok := media.Document.AsNotEmpty()
		if !ok {
			return drive.ChannelFile{}, false
		}
		name := ""
		for _, attr := range doc.Attributes {
			if fn, ok := attr.(*tg.DocumentAttributeFilename); ok && fn.FileName != "" {
				name = fn.FileName
				break
			}
		}
		if name == "" {
			name = fmt.Sprintf("telegram-%d%s", msg.ID, extFromMime(doc.MimeType))
		}
		return drive.ChannelFile{MessageID: msg.ID, Name: name, Size: doc.Size, MimeType: doc.MimeType}, true
	case *tg.MessageMediaPhoto:
		photo, ok := media.Photo.AsNotEmpty()
		if !ok {
			return drive.ChannelFile{}, false
		}
		var size int64
		for _, ps := range photo.Sizes {
			if s, ok := ps.(*tg.PhotoSize); ok && int64(s.Size) > size {
				size = int64(s.Size)
			}
		}
		return drive.ChannelFile{MessageID: msg.ID, Name: fmt.Sprintf("telegram-photo-%d.jpg", msg.ID), Size: size, MimeType: "image/jpeg"}, true
	}
	return drive.ChannelFile{}, false
}

func extFromMime(mime string) string {
	switch mime {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg":
		return ".mp3"
	case "application/zip":
		return ".zip"
	}
	return ".bin"
}

