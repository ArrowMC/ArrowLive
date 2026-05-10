package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"ArrowLiveWebsite/internal/config"
	"ArrowLiveWebsite/internal/storage"
)

var (
	ErrInvalidName = errors.New("invalid room name")
	ErrOccupied    = errors.New("room name is currently occupied")
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

type Info struct {
	Name    string `json:"name"`
	Token   string `json:"token"`
	RTMPURL string `json:"rtmp_url"`
	PlayURL string `json:"play_url"`
	Status  string `json:"status"`
}

type Service struct {
	store *storage.Store
	cfg   *config.ZLMediaKit
}

func New(store *storage.Store, cfg *config.ZLMediaKit) *Service {
	return &Service{store: store, cfg: cfg}
}

func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

func (s *Service) Create(name string) (*Info, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	existing, err := s.store.Get(name)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.Status != storage.StatusInactive {
		return nil, ErrOccupied
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertNew(name, token); err != nil {
		return nil, err
	}
	return s.buildInfo(name, token, storage.StatusCreated), nil
}

func (s *Service) Status(name string) (*Info, error) {
	r, err := s.store.Get(name)
	if err != nil {
		return nil, err
	}
	return s.buildInfo(r.Name, r.Token, r.Status), nil
}

func (s *Service) Get(name string) (*storage.Room, error) {
	return s.store.Get(name)
}

func (s *Service) SetStatus(name, status string) error {
	return s.store.SetStatus(name, status)
}

func (s *Service) ListActive() ([]*storage.Room, error) {
	return s.store.ListActive()
}

func (s *Service) buildInfo(name, token, status string) *Info {
	return &Info{
		Name:    name,
		Token:   token,
		Status:  status,
		RTMPURL: fmt.Sprintf("%s/%s/%s?token=%s", s.cfg.RTMPPushBase, s.cfg.App, name, token),
		PlayURL: fmt.Sprintf("%s/%s/%s.live.ts", s.cfg.HTTPFLVPlayBase, s.cfg.App, name),
	}
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
