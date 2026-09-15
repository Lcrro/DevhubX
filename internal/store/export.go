package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lcrro/DevhubX/internal/model"
)

type Bundle struct {
	App           string                 `json:"app"`
	Format        int                    `json:"format"`
	ExportedAt    string                 `json:"exportedAt"`
	SchemaVersion int                    `json:"schemaVersion"`
	Settings      model.Settings         `json:"settings"`
	Services      []model.Service        `json:"services"`
	Logs          map[string][]model.Log `json:"logs,omitempty"`
	Stripped      []string               `json:"stripped"`
}

type ExportOptions struct {
	IncludeLogs    bool
	IncludeSecrets bool
}

func SanitizeImported(v model.Service) model.Service {
	v.PID = 0
	v.Birth = 0
	v.Managed = false
	v.Status = "stopped"
	v.Cover = ""
	v.ScreenshotError = ""
	v.Health = model.Health{}
	v.Failure = ""
	v.FailureDetail = ""
	v.Env = model.NormalizeEnv(v.Env)
	return v
}

func (s *Store) Export(opts ExportOptions) (Bundle, error) {
	settings, err := s.LoadSettings()
	if err != nil {
		return Bundle{}, err
	}
	services, err := s.List()
	if err != nil {
		return Bundle{}, err
	}
	stripped := []string{"runtime", "covers"}
	out := make([]model.Service, 0, len(services))
	logs := map[string][]model.Log{}
	for _, v := range services {
		v = SanitizeImported(v)
		if !opts.IncludeSecrets {
			v = v.Redacted()
		}
		out = append(out, v)
		if opts.IncludeLogs {
			entries, err := s.Logs(v.ID, 0)
			if err != nil {
				return Bundle{}, err
			}
			if len(entries) > 0 {
				logs[v.ID] = entries
			}
		}
	}
	if !opts.IncludeSecrets {
		stripped = append(stripped, "secrets")
	}
	if !opts.IncludeLogs {
		stripped = append(stripped, "logs")
	}
	bundle := Bundle{
		App:           "devhub",
		Format:        1,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		SchemaVersion: SchemaVersion,
		Settings:      settings.Normalized(),
		Services:      out,
		Stripped:      stripped,
	}
	if opts.IncludeLogs {
		bundle.Logs = logs
	}
	return bundle, nil
}

func ValidateBundle(bundle Bundle) error {
	if bundle.App != "devhub" {
		return fmt.Errorf("不是 DevHub 配置包")
	}
	if bundle.Format != 1 {
		return fmt.Errorf("不支持的导出格式")
	}
	bundle.Settings = bundle.Settings.Normalized()
	if err := bundle.Settings.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, v := range bundle.Services {
		if v.ID == "" || seen[v.ID] {
			return fmt.Errorf("配置包包含无效或重复的服务 ID")
		}
		seen[v.ID] = true
		if v.Name == "" || v.Port < 1 || v.Port > 65535 {
			return fmt.Errorf("配置包包含无效服务")
		}
	}
	return nil
}

func (s *Store) ReplaceAll(settings model.Settings, services []model.Service, logs map[string][]model.Log) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM logs`); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM services`); err != nil {
		return err
	}
	for _, v := range services {
		v = SanitizeImported(v)
		v.Updated = time.Now().UTC().Format(time.RFC3339)
		b, err := jsonMarshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO services(id,payload) VALUES(?,?)`, v.ID, string(b)); err != nil {
			return err
		}
		for _, entry := range logs[v.ID] {
			text := entry.Text
			if len(text) > 16384 {
				text = text[len(text)-16384:]
			}
			when := entry.Time
			if when == "" {
				when = v.Updated
			}
			if _, err = tx.Exec(`INSERT INTO logs(service_id,time,text) VALUES(?,?,?)`, v.ID, when, text); err != nil {
				return err
			}
		}
	}
	settings = settings.Normalized()
	raw, err := jsonMarshal(settings)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO settings(key,value) VALUES('global',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Backup(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	_, err := s.db.Exec(`VACUUM INTO ?`, path)
	return err
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
