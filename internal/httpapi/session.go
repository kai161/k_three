package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"unwind/internal/plan"
)

const sessionCookieName = "unwind_session"

// SessionManager issues anonymous, opaque browser sessions.
type SessionManager struct {
	store  plan.Store
	secure bool
}

func NewSessionManager(store plan.Store, secure bool) *SessionManager {
	return &SessionManager{store: store, secure: secure}
}

func (m *SessionManager) Ensure(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil && len(decoded) == 32 {
			deviceID := hashToken(cookie.Value)
			if err := m.store.EnsureDevice(r.Context(), deviceID); err != nil {
				return "", err
			}
			return deviceID, nil
		}
	}

	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(token[:])
	deviceID := hashToken(value)
	if err := m.store.EnsureDevice(r.Context(), deviceID); err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 60 * 60,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	})
	return deviceID, nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
