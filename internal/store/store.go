package store

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Lcrro/DevhubX/internal/model"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db}
	if err = s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) LoadSettings() (model.Settings, error) {
	settings := model.DefaultSettings()
	var raw string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key='global'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return model.DefaultSettings(), err
	}
	return settings.Normalized(), nil
}

func (s *Store) SaveSettings(settings model.Settings) error {
	settings = settings.Normalized()
	b, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO settings(key,value) VALUES('global',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(b))
	return err
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Save(v model.Service) error {
	v.Updated = time.Now().UTC().Format(time.RFC3339)
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO services(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload`, v.ID, string(b))
	return err
}
func (s *Store) List() ([]model.Service, error) {
	rows, err := s.db.Query(`SELECT payload FROM services ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Service{}
	for rows.Next() {
		var b string
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		var v model.Service
		if err = json.Unmarshal([]byte(b), &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (s *Store) Delete(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM logs WHERE service_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM services WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Append(id, text string) error {
	if len(text) > 16384 {
		text = text[len(text)-16384:]
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO logs(service_id,time,text) VALUES(?,?,?)`, id, time.Now().UTC().Format(time.RFC3339), text); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM logs WHERE service_id=? AND id NOT IN (SELECT id FROM logs WHERE service_id=? ORDER BY id DESC LIMIT 500)`, id, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Logs(id string, after int64) ([]model.Log, error) {
	rows, err := s.db.Query(`SELECT id,time,text FROM logs WHERE service_id=? AND id>? ORDER BY id LIMIT 500`, id, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Log{}
	for rows.Next() {
		var l model.Log
		if err = rows.Scan(&l.ID, &l.Time, &l.Text); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

func (s *Store) ClearLogs(id string) error {
	_, err := s.db.Exec(`DELETE FROM logs WHERE service_id=?`, id)
	return err
}
