package api

import (
	"net/http"

	"golang.org/x/net/webdav"

	"telegram-drive-agent/internal/webdavfs"
)

func (s *Server) webdavHandler() http.Handler {
	handler := &webdav.Handler{
		Prefix:     "/webdav",
		FileSystem: webdavfs.New(s.drive),
		LockSystem: webdav.NewMemLS(),
	}
	return handler
}
