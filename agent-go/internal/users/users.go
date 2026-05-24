package users

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("email hoặc mật khẩu không đúng")
	ErrEmailExists        = errors.New("email đã tồn tại")
	ErrSessionInvalid     = errors.New("phiên đăng nhập không hợp lệ")
)

const sessionCookie = "td_session"
const sessionTTL = 30 * 24 * time.Hour

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role"`
	CreatedAt   int64  `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) HasAnyUser(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) Create(ctx context.Context, email, password, displayName, role string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return User{}, errors.New("email trống")
	}
	if len(password) < 6 {
		return User{}, errors.New("mật khẩu tối thiểu 6 ký tự")
	}
	if role == "" {
		role = "admin"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	now := time.Now().Unix()
	id := newID()
	_, err = s.db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at) VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?)`, id, email, string(hash), displayName, role, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	return User{ID: id, Email: email, DisplayName: displayName, Role: role, CreatedAt: now}, nil
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash, COALESCE(display_name, ''), role, created_at FROM users WHERE email = ?`, email)
	var user User
	var hash string
	if err := row.Scan(&user.ID, &user.Email, &hash, &user.DisplayName, &user.Role, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) CreateSession(ctx context.Context, userID, userAgent string) (string, time.Time, error) {
	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	created := time.Now()
	expires := created.Add(sessionTTL)
	_, err = s.db.ExecContext(ctx, `INSERT INTO user_sessions (token, user_id, created_at, expires_at, user_agent) VALUES (?, ?, ?, ?, NULLIF(?, ''))`, token, userID, created.Unix(), expires.Unix(), userAgent)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (s *Service) ResolveSession(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrSessionInvalid
	}
	row := s.db.QueryRowContext(ctx, `SELECT u.id, u.email, COALESCE(u.display_name, ''), u.role, u.created_at FROM user_sessions s JOIN users u ON u.id = s.user_id WHERE s.token = ? AND s.expires_at > ?`, token, time.Now().Unix())
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrSessionInvalid
		}
		return User{}, err
	}
	return user, nil
}

func (s *Service) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
	return err
}

func WriteSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

func TokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		return cookie.Value
	}
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return ""
}

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("u-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
