package storage

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("link not found")
	ErrConflict = errors.New("code already exists")
)

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type Link struct {
	ID          int64      `json:"-"`
	Code        string     `json:"code"`
	URL         string     `json:"url"`
	Visits      int        `json:"visits"`
	LastVisitAt *time.Time `json:"last_visit_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Storage struct {
	db *sql.DB
}

func New(dbPath string) (*Storage, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

func (s *Storage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL,
		visits INTEGER NOT NULL DEFAULT 0,
		last_visit_at DATETIME NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_links_code ON links(code);
	`
	_, err := s.db.Exec(query)
	return err
}

func GenerateCode(length int) (string, error) {
	ret := make([]byte, length)
	charLen := big.NewInt(int64(len(base62Chars)))
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charLen)
		if err != nil {
			return "", err
		}
		ret[i] = base62Chars[num.Int64()]
	}
	return string(ret), nil
}

func (s *Storage) CreateLink(targetURL, customCode string) (*Link, error) {
	code := customCode
	if code == "" {
		for i := 0; i < 5; i++ {
			c, err := GenerateCode(7)
			if err != nil {
				return nil, fmt.Errorf("generate code: %w", err)
			}
			link, err := s.insertLink(targetURL, c)
			if err == nil {
				return link, nil
			}
			if !errors.Is(err, ErrConflict) {
				return nil, err
			}
		}
		return nil, fmt.Errorf("failed to generate unique code after retries")
	}

	return s.insertLink(targetURL, code)
}

func (s *Storage) insertLink(targetURL, code string) (*Link, error) {
	query := `
	INSERT INTO links (code, url, visits, created_at)
	VALUES (?, ?, 0, CURRENT_TIMESTAMP)
	RETURNING id, code, url, visits, last_visit_at, created_at;
	`
	var link Link
	err := s.db.QueryRow(query, code, targetURL).Scan(
		&link.ID,
		&link.Code,
		&link.URL,
		&link.Visits,
		&link.LastVisitAt,
		&link.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert link: %w", err)
	}
	return &link, nil
}

func (s *Storage) Reset() error {
	_, err := s.db.Exec(`DELETE FROM links; VACUUM;`)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}