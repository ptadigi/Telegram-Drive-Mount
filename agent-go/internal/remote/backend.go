package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"telegram-drive-agent/internal/drive"
)

// Backend implements vfs.Backend by talking HTTPS to a Telegram Drive VPS.
// Auth uses the device token minted via td-agent --pair.
type Backend struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewBackend(baseURL, token string, timeout time.Duration) *Backend {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Backend{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (b *Backend) request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(b.baseURL + path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Device "+b.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (b *Backend) doJSON(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := b.request(ctx, method, path, rdr)
	if err != nil {
		return err
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errResp)
		if errResp.Error != "" {
			return fmt.Errorf("remote %s %s: %s", method, path, errResp.Error)
		}
		return fmt.Errorf("remote %s %s: HTTP %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (b *Backend) ListFolderContents(ctx context.Context, folderID string) (drive.FolderContents, error) {
	var out drive.FolderContents
	path := "/v1/drive/contents?folder_id=" + url.QueryEscape(folderID)
	if err := b.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return drive.FolderContents{}, err
	}
	return out, nil
}

func (b *Backend) ResolveFolderByPath(ctx context.Context, p string) (string, error) {
	clean := strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if clean == "" {
		return "", nil
	}
	current := ""
	for _, segment := range strings.Split(clean, "/") {
		contents, err := b.ListFolderContents(ctx, current)
		if err != nil {
			return "", err
		}
		next := ""
		for _, folder := range contents.Folders {
			if folder.Name == segment {
				next = folder.ID
				break
			}
		}
		if next == "" {
			return "", fmt.Errorf("khong tim thay thu muc %q", segment)
		}
		current = next
	}
	return current, nil
}

func (b *Backend) ResolveFileByPath(ctx context.Context, p string) (drive.File, error) {
	clean := strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if clean == "" {
		return drive.File{}, errors.New("duong dan trong")
	}
	idx := strings.LastIndex(clean, "/")
	parent := ""
	leaf := clean
	if idx >= 0 {
		parent = clean[:idx]
		leaf = clean[idx+1:]
	}
	folderID, err := b.ResolveFolderByPath(ctx, parent)
	if err != nil {
		return drive.File{}, err
	}
	contents, err := b.ListFolderContents(ctx, folderID)
	if err != nil {
		return drive.File{}, err
	}
	for _, file := range contents.Files {
		if file.Name == leaf {
			return file, nil
		}
	}
	return drive.File{}, fmt.Errorf("khong tim thay file %q", leaf)
}

func (b *Backend) ResolveFolderEntryByPath(ctx context.Context, p string) (drive.Folder, error) {
	clean := strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if clean == "" {
		return drive.Folder{}, errors.New("duong dan trong")
	}
	idx := strings.LastIndex(clean, "/")
	parent := ""
	leaf := clean
	if idx >= 0 {
		parent = clean[:idx]
		leaf = clean[idx+1:]
	}
	folderID, err := b.ResolveFolderByPath(ctx, parent)
	if err != nil {
		return drive.Folder{}, err
	}
	contents, err := b.ListFolderContents(ctx, folderID)
	if err != nil {
		return drive.Folder{}, err
	}
	for _, folder := range contents.Folders {
		if folder.Name == leaf {
			return folder, nil
		}
	}
	return drive.Folder{}, fmt.Errorf("khong tim thay thu muc %q", leaf)
}

func (b *Backend) StreamFromTelegram(ctx context.Context, fileID string, offset, length int64, w io.Writer) (drive.StreamResult, error) {
	req, err := b.request(ctx, http.MethodGet, "/v1/files/stream?id="+url.QueryEscape(fileID), nil)
	if err != nil {
		return drive.StreamResult{}, err
	}
	if length > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return drive.StreamResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return drive.StreamResult{}, fmt.Errorf("remote stream HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		return drive.StreamResult{}, err
	}
	out := drive.StreamResult{Size: written, MimeType: resp.Header.Get("Content-Type")}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if n, _ := strconv.ParseInt(cl, 10, 64); n > 0 {
			out.Size = n
		}
	}
	return out, nil
}

func (b *Backend) SaveStreamFile(ctx context.Context, data io.Reader, filename, mimeHint, folderID, relativePath string) (drive.File, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer mw.Close()
		if folderID != "" {
			_ = mw.WriteField("folder_id", folderID)
		}
		if relativePath != "" {
			_ = mw.WriteField("relative_path", relativePath)
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
		if mimeHint != "" {
			header.Set("Content-Type", mimeHint)
		}
		part, err := mw.CreatePart(header)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, data); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	endpoint := b.baseURL + "/v1/files/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, pr)
	if err != nil {
		return drive.File{}, err
	}
	req.Header.Set("Authorization", "Device "+b.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := b.http.Do(req)
	if err != nil {
		return drive.File{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errResp)
		if errResp.Error != "" {
			return drive.File{}, fmt.Errorf("remote upload: %s", errResp.Error)
		}
		return drive.File{}, fmt.Errorf("remote upload HTTP %d", resp.StatusCode)
	}
	var result struct {
		File drive.File `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return drive.File{}, fmt.Errorf("decode upload response: %w", err)
	}
	return result.File, nil
}

func (b *Backend) CreateFolder(ctx context.Context, input drive.CreateFolderInput) (drive.Folder, error) {
	var resp struct {
		Folder drive.Folder `json:"folder"`
	}
	if err := b.doJSON(ctx, http.MethodPost, "/v1/folders", input, &resp); err != nil {
		return drive.Folder{}, err
	}
	return resp.Folder, nil
}

func (b *Backend) TrashFile(ctx context.Context, id string) error {
	return b.doJSON(ctx, http.MethodPost, "/v1/files/trash", map[string]string{"id": id}, nil)
}

func (b *Backend) TrashFolder(ctx context.Context, id string) error {
	return b.doJSON(ctx, http.MethodPost, "/v1/folders/trash", map[string]string{"id": id}, nil)
}

func (b *Backend) RenameFile(ctx context.Context, input drive.RenameInput) (drive.File, error) {
	var resp struct {
		File drive.File `json:"file"`
	}
	if err := b.doJSON(ctx, http.MethodPut, "/v1/files/rename", input, &resp); err != nil {
		return drive.File{}, err
	}
	return resp.File, nil
}

func (b *Backend) RenameFolder(ctx context.Context, input drive.RenameInput) (drive.Folder, error) {
	var resp struct {
		Folder drive.Folder `json:"folder"`
	}
	if err := b.doJSON(ctx, http.MethodPut, "/v1/folders/rename", input, &resp); err != nil {
		return drive.Folder{}, err
	}
	return resp.Folder, nil
}

func (b *Backend) MoveFile(ctx context.Context, input drive.MoveInput) (drive.File, error) {
	var resp struct {
		File drive.File `json:"file"`
	}
	if err := b.doJSON(ctx, http.MethodPut, "/v1/files/move", input, &resp); err != nil {
		return drive.File{}, err
	}
	return resp.File, nil
}

func (b *Backend) MoveFolder(ctx context.Context, input drive.MoveInput) (drive.Folder, error) {
	var resp struct {
		Folder drive.Folder `json:"folder"`
	}
	if err := b.doJSON(ctx, http.MethodPut, "/v1/folders/move", input, &resp); err != nil {
		return drive.Folder{}, err
	}
	return resp.Folder, nil
}