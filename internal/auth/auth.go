package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"ArrowLiveWebsite/internal/config"
)

const (
	cookieName = "admin_session"
	sessionTTL = 12 * time.Hour
)

type Manager struct {
	cfg      *config.Admin
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func NewManager(cfg *config.Admin) *Manager {
	m := &Manager{cfg: cfg, sessions: make(map[string]time.Time)}
	go m.gc()
	return m
}

func (m *Manager) Check(username, password string) bool {
	uOK := subtle.ConstantTimeCompare([]byte(username), []byte(m.cfg.Username)) == 1
	pOK := subtle.ConstantTimeCompare([]byte(password), []byte(m.cfg.Password)) == 1
	return uOK && pOK
}

func (m *Manager) Issue(w http.ResponseWriter) error {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	sid := hex.EncodeToString(b)
	m.mu.Lock()
	m.sessions[sid] = time.Now().Add(sessionTTL)
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return nil
}

func (m *Manager) Revoke(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName)
	if err == nil {
		m.mu.Lock()
		delete(m.sessions, c.Value)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (m *Manager) Valid(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	m.mu.RLock()
	exp, ok := m.sessions[c.Value]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		m.mu.Lock()
		delete(m.sessions, c.Value)
		m.mu.Unlock()
		return false
	}
	return true
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.Valid(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Manager) gc() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		m.mu.Lock()
		for k, v := range m.sessions {
			if now.After(v) {
				delete(m.sessions, k)
			}
		}
		m.mu.Unlock()
	}
}
