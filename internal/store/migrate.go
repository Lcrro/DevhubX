package store

import (
	"database/sql"
	"fmt"
)

const SchemaVersion = 3

func (s *Store) migrate() error {
	version, err := s.UserVersion()
	if err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("数据库由更新的 DevHub 创建（schema %d），当前支持 %d", version, SchemaVersion)
	}
	for version < SchemaVersion {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		next := version + 1
		if err = apply(tx, next); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.Exec(fmt.Sprintf(`PRAGMA user_version=%d`, next)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		version = next
	}
	return nil
}

func apply(tx *sql.Tx, version int) error {
	switch version {
	case 1:
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS services (id TEXT PRIMARY KEY, payload TEXT NOT NULL);`)
		return err
	case 2:
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS logs (id INTEGER PRIMARY KEY AUTOINCREMENT, service_id TEXT NOT NULL, time TEXT NOT NULL, text TEXT NOT NULL);
			CREATE INDEX IF NOT EXISTS logs_service ON logs(service_id, id);
			CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`)
		return err
	case 3:
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
			INSERT INTO schema_meta(key,value) VALUES('app','devhub') ON CONFLICT(key) DO UPDATE SET value=excluded.value;`)
		return err
	default:
		return fmt.Errorf("未知 schema 版本 %d", version)
	}
}

func (s *Store) UserVersion() (int, error) {
	var version int
	err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version)
	return version, err
}
