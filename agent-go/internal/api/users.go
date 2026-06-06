package api

import (
	"net/http"

	"telegram-drive-agent/internal/users"
)

func (s *Server) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusInternalServerError, errBadRequest("user service chưa sẵn sàng"))
		return
	}
	var input struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, token, expires, err := s.users.RegisterFirstAdmin(r.Context(), input.Email, input.Password, input.DisplayName, r.Header.Get("User-Agent"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.drive.AdoptOrphanedData(r.Context(), user.ID); err != nil {
		s.drive.WriteAudit(r.Context(), user.ID, "user.adopt_failed", "user", user.ID, map[string]any{"error": err.Error()})
	}
	users.WriteSessionCookie(w, r, token, expires)
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusInternalServerError, errBadRequest("user service chưa sẵn sàng"))
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := s.users.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	token, expires, err := s.users.CreateSession(r.Context(), user.ID, r.Header.Get("User-Agent"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	users.WriteSessionCookie(w, r, token, expires)
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleUserLogout(w http.ResponseWriter, r *http.Request) {
	if s.users != nil {
		_ = s.users.DeleteSession(r.Context(), users.TokenFromRequest(r))
	}
	users.ClearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleUserMe(w http.ResponseWriter, r *http.Request) {
	if s.users == nil {
		writeError(w, http.StatusInternalServerError, errBadRequest("user service chưa sẵn sàng"))
		return
	}
	has, err := s.users.HasAnyUser(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !has {
		writeJSON(w, http.StatusOK, map[string]any{"setup": true})
		return
	}
	user, err := s.users.ResolveSession(r.Context(), users.TokenFromRequest(r))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"setup": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "setup": false})
}
