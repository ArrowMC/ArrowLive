package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const (
	StatusCreated  = "created"
	StatusActive   = "active"
	StatusInactive = "inactive"
)

var ErrNotFound = errors.New("room not found")

type Room struct {
	Name      string
	Token     string
	Status    string
	CreatedAt int64
	ActiveAt  int64
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			name       TEXT PRIMARY KEY,
			token      TEXT NOT NULL,
			status     TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			active_at  INTEGER NOT NULL DEFAULT 0
		);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Get(name string) (*Room, error) {
	row := s.db.QueryRow(`SELECT name, token, status, created_at, active_at FROM rooms WHERE name = ?`, name)
	var r Room
	if err := row.Scan(&r.Name, &r.Token, &r.Status, &r.CreatedAt, &r.ActiveAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) UpsertNew(name, token string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO rooms(name, token, status, created_at, active_at)
		VALUES(?, ?, ?, ?, 0)
		ON CONFLICT(name) DO UPDATE SET
			token = excluded.token,
			status = excluded.status,
			created_at = excluded.created_at,
			active_at = 0
	`, name, token, StatusCreated, now)
	return err
}

func (s *Store) SetStatus(name, status string) error {
	now := time.Now().Unix()
	if status == StatusActive {
		_, err := s.db.Exec(`UPDATE rooms SET status = ?, active_at = ? WHERE name = ?`, status, now, name)
		return err
	}
	_, err := s.db.Exec(`UPDATE rooms SET status = ? WHERE name = ?`, status, name)
	return err
}

func (s *Store) ListActive() ([]*Room, error) {
	rows, err := s.db.Query(`SELECT name, token, status, created_at, active_at FROM rooms WHERE status = ? ORDER BY created_at ASC`, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.Name, &r.Token, &r.Status, &r.CreatedAt, &r.ActiveAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// ResetOccupied 删除所有处于 created 或 active 的房间记录。
// 服务重启期间 ZLM 的 on_stream_changed 回调可能丢失，这些状态不再可信，
// 清掉能避免幽灵占用阻止同名房间被重新领用。
func (s *Store) ResetOccupied() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM rooms WHERE status IN (?, ?)`, StatusCreated, StatusActive)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
