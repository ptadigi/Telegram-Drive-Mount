package api

import (
	"net/http"
)

// handlePhotosMissing accepts a batch of SHA-256 hashes and returns the subset
// the Drive does NOT already have. The PWA photo-backup uses this to upload
// only new photos (incremental, dedup) instead of re-sending the whole album.
func (s *Server) handlePhotosMissing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Hashes []string `json:"hashes"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	present, err := s.drive.PhotoHashesPresent(r.Context(), input.Hashes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	missing := make([]string, 0)
	seen := make(map[string]struct{})
	for _, h := range input.Hashes {
		key := normalizeHash(h)
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if !present[key] {
			missing = append(missing, key)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"missing": missing})
}

// handlePhotosUpload saves one photo into the per-user "Camera" folder and
// queues it for Telegram sync. Multipart: field "file". The folder is created
// on demand. Runs behind withAuth (session or device token).
func (s *Server) handlePhotosUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	camera, err := s.drive.EnsureCameraFolder(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	saved, err := s.drive.SaveStreamFile(r.Context(), file, header.Filename, header.Header.Get("Content-Type"), camera.ID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": saved, "folder_id": camera.ID})
}

func normalizeHash(h string) string {
	out := make([]byte, 0, len(h))
	for i := 0; i < len(h); i++ {
		c := h[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			out = append(out, c)
		}
	}
	return string(out)
}
