package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"devhub/internal/model"
)

func TestPersistenceAndLogRetention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	v := model.Service{ID: "test", Name: "中文项目", Port: 3000, Command: "npm run dev"}
	if err = s.Save(v); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 505; i++ {
		if err = s.Append(v.ID, "hello\n"); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	all, err := s.List()
	if err != nil || len(all) != 1 || all[0].Name != v.Name {
		t.Fatalf("persistence: %v %v", all, err)
	}
	logs, err := s.Logs(v.ID, 0)
	if err != nil || len(logs) != 500 {
		t.Fatalf("retention: %d %v", len(logs), err)
	}
	next, err := s.Logs(v.ID, logs[498].ID)
	if err != nil || len(next) != 1 {
		t.Fatal("incremental logs failed", err)
	}
	if err = s.Delete(v.ID); err != nil {
		t.Fatal(err)
	}
	logs, _ = s.Logs(v.ID, 0)
	if len(logs) != 0 {
		t.Fatal("orphan logs")
	}
}

func TestSettingsPersistenceAndNormalization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadSettings()
	if err != nil || got.Language != "zh" || !got.AutoScan || got.ScanIntervalSeconds != 10 {
		t.Fatalf("defaults: %+v %v", got, err)
	}
	if err := s.SaveSettings(model.Settings{
		Language:            "en",
		AutoScan:            false,
		ScanIntervalSeconds: 999,
		IncludeDirectories:  []string{t.TempDir(), "relative/path"},
		ExcludeProcesses:    []string{"QQ.exe", "qq.exe", ""},
		ExcludePorts:        []int{4301, 4301, 0},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err = s.LoadSettings()
	if err != nil || got.Language != "en" || got.AutoScan || got.ScanIntervalSeconds != model.MaxScanIntervalSeconds {
		t.Fatalf("persisted settings: %+v %v", got, err)
	}
	if len(got.IncludeDirectories) != 1 || !filepath.IsAbs(got.IncludeDirectories[0]) {
		t.Fatalf("include directories: %+v", got.IncludeDirectories)
	}
	if len(got.ExcludeProcesses) != 1 || got.ExcludeProcesses[0] != "QQ.exe" {
		t.Fatalf("exclude processes: %+v", got.ExcludeProcesses)
	}
	if len(got.ExcludePorts) != 1 || got.ExcludePorts[0] != 4301 {
		t.Fatalf("exclude ports: %+v", got.ExcludePorts)
	}
}

func TestClearLogs(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "clear.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Append("service", "one\n"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearLogs("service"); err != nil {
		t.Fatal(err)
	}
	logs, err := s.Logs("service", 0)
	if err != nil || len(logs) != 0 {
		t.Fatalf("clear logs: %d %v", len(logs), err)
	}
}

func TestMigrateFromV1AndUnknownVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE services (id TEXT PRIMARY KEY, payload TEXT NOT NULL);
		INSERT INTO services(id,payload) VALUES('old','{"id":"old","name":"legacy","port":3000}');
		PRAGMA user_version=1;`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	version, err := s.UserVersion()
	if err != nil || version != SchemaVersion {
		t.Fatalf("migrated version: %d %v", version, err)
	}
	all, err := s.List()
	if err != nil || len(all) != 1 || all[0].Name != "legacy" {
		t.Fatalf("migrated services: %+v %v", all, err)
	}
	s.Close()

	newer := filepath.Join(t.TempDir(), "new.db")
	db, err = sql.Open("sqlite", newer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE services (id TEXT PRIMARY KEY, payload TEXT NOT NULL); PRAGMA user_version=99;`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err = Open(newer); err == nil {
		t.Fatal("newer schema was accepted")
	}
}

func TestExportImportRoundTripAndSecrets(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "exp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := model.Service{
		ID:   "svc",
		Name: "api",
		Port: 8080,
		Env:  []model.EnvVar{{Name: "API_TOKEN", Value: "super-secret", Secret: true}, {Name: "PORT", Value: "8080"}},
	}
	if err = s.Save(v); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveSettings(model.Settings{Language: "en", AutoScan: false, ScanIntervalSeconds: 30, Theme: "light"}); err != nil {
		t.Fatal(err)
	}
	if err = s.Append("svc", "log-line\n"); err != nil {
		t.Fatal(err)
	}
	stripped, err := s.Export(ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if stripped.Logs != nil {
		t.Fatal("logs included by default")
	}
	found := stripped.Services[0]
	if len(found.Env) != 2 || found.Env[0].Value != "" || found.Env[1].Value != "8080" {
		t.Fatalf("secrets not stripped: %+v", found.Env)
	}
	full, err := s.Export(ExportOptions{IncludeLogs: true, IncludeSecrets: true})
	if err != nil || full.Services[0].Env[0].Value != "super-secret" || len(full.Logs["svc"]) == 0 {
		t.Fatalf("full export: %+v %v", full, err)
	}

	empty, err := Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	if err = empty.ReplaceAll(stripped.Settings, stripped.Services, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := empty.List()
	settings, _ := empty.LoadSettings()
	if len(got) != 1 || got[0].Name != "api" || settings.Language != "en" || settings.Theme != "light" {
		t.Fatalf("imported: %+v %+v", got, settings)
	}
	backup := filepath.Join(t.TempDir(), "backups", "copy.db")
	if err = s.Backup(backup); err != nil {
		t.Fatal(err)
	}
	copied, err := Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	copied.Close()
}

func TestImportFailureLeavesOriginal(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "keep.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Save(model.Service{ID: "keep", Name: "keep-me", Port: 1}); err != nil {
		t.Fatal(err)
	}
	if err = ValidateBundle(Bundle{App: "other", Format: 1}); err == nil {
		t.Fatal("invalid bundle accepted")
	}
	all, _ := s.List()
	if len(all) != 1 || all[0].Name != "keep-me" {
		t.Fatalf("original changed: %+v", all)
	}
}
