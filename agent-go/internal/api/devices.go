package api

import (
	"errors"
	"net/http"
	"runtime"
	"strings"

	"telegram-drive-agent/internal/devices"
	"telegram-drive-agent/internal/drive"
)

const deviceAuthScheme = "Device "

// resolveDeviceUser inspects the Authorization header for a device token.
// Returns the resolved user_id when valid, or empty string when the header
// is missing/invalid.
func (s *Server) resolveDeviceUser(r *http.Request) (string, *devices.Device) {
	if s.devices == nil {
		return "", nil
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, deviceAuthScheme) {
		return "", nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, deviceAuthScheme))
	if token == "" {
		return "", nil
	}
	dev, err := s.devices.ResolveToken(r.Context(), token)
	if err != nil {
		return "", nil
	}
	return dev.UserID, &dev
}

// handleDevicePairStart is called by an authenticated PWA session to mint a
// short pairing code that a remote td-agent client will type in.
func (s *Server) handleDevicePairStart(w http.ResponseWriter, r *http.Request) {
	if s.devices == nil {
		writeError(w, http.StatusServiceUnavailable, errBadRequest("device service chua san sang"))
		return
	}
	userID := drive.UserFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, errors.New("can dang nhap PWA truoc khi tao ma ghep thiet bi"))
		return
	}
	pc, err := s.devices.StartPairing(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, pc)
}

// handleDevicePairExchange is hit by td-agent --pair without a session cookie;
// it trades the human-typed code + device metadata for a long-lived token.
// Rate-limited per IP via shareRate (existing limiter is fine for our scale).
func (s *Server) handleDevicePairExchange(w http.ResponseWriter, r *http.Request) {
	if s.devices == nil {
		writeError(w, http.StatusServiceUnavailable, errBadRequest("device service chua san sang"))
		return
	}
	if !s.shareRate.allow("pair:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, errBadRequest("qua nhieu yeu cau, vui long thu lai sau"))
		return
	}
	var input struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Platform string `json:"platform"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = runtime.GOOS
	}
	res, err := s.devices.ExchangeCode(r.Context(), input.Code, input.Name, platform, clientIP(r))
	if err != nil {
		switch err {
		case devices.ErrPairingNotFound, devices.ErrPairingExpired, devices.ErrPairingConsumed, devices.ErrInvalidPayload:
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDeviceList(w http.ResponseWriter, r *http.Request) {
	if s.devices == nil {
		writeError(w, http.StatusServiceUnavailable, errBadRequest("device service chua san sang"))
		return
	}
	userID := drive.UserFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, errors.New("can dang nhap PWA"))
		return
	}
	list, err := s.devices.ListDevices(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": list})
}

func (s *Server) handleDeviceRevoke(w http.ResponseWriter, r *http.Request) {
	if s.devices == nil {
		writeError(w, http.StatusServiceUnavailable, errBadRequest("device service chua san sang"))
		return
	}
	userID := drive.UserFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, errors.New("can dang nhap PWA"))
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thieu id thiet bi"))
		return
	}
	if err := s.devices.RevokeDevice(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}