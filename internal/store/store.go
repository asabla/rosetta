package store

import (
	"database/sql"
	"encoding/json"
	"github.com/asabla/rosetta/internal/compiler"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db}
	return s, s.init()
}
func (s *Store) init() error {
	_, err := s.DB.Exec(`create table if not exists sandboxes(id text primary key, principal text, policy_path text, policy_hash text, grants text, created_at text)`)
	return err
}
func (s *Store) Create(id, principal, path, hash string, grants []compiler.Grant) error {
	b, _ := json.Marshal(grants)
	_, err := s.DB.Exec(`insert into sandboxes values(?,?,?,?,?,?)`, id, principal, path, hash, string(b), time.Now().UTC().Format(time.RFC3339))
	return err
}
func (s *Store) Get(id string) (string, string, []compiler.Grant, error) {
	var p, h, g string
	err := s.DB.QueryRow(`select policy_path, policy_hash, grants from sandboxes where id=?`, id).Scan(&p, &h, &g)
	var grants []compiler.Grant
	_ = json.Unmarshal([]byte(g), &grants)
	return p, h, grants, err
}
func (s *Store) Update(id, path, hash string, grants []compiler.Grant) error {
	b, _ := json.Marshal(grants)
	_, err := s.DB.Exec(`update sandboxes set policy_path=?, policy_hash=?, grants=? where id=?`, path, hash, string(b), id)
	return err
}
