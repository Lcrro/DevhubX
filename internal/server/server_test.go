package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Lcrro/DevhubX/internal/model"
)

func TestLoopbackOnly(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:4780", "localhost:4780", "192.168.1.2:4780", "127.0.0.1:0", "127.0.0.1:70000"} {
		if _, err := ListenAddress(addr); err == nil {
			t.Error("accepted", addr)
		}
	}
	if _, err := ListenAddress("127.0.0.1:4780"); err != nil {
		t.Fatal(err)
	}
}
func TestSecurityAndCRUD(t *testing.T) {
	s, err := New(t.TempDir(), 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h := s.Handler(os.DirFS(t.TempDir()))
	request := func(method, path, host, origin, token string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "http://"+host+path, bytes.NewReader(b))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-DevHub-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	for _, tc := range []struct{ host, origin, token string }{{"evil.test:4780", "", ""}, {"127.0.0.1:4780", "https://evil.test", s.token}, {"127.0.0.1:4780", "", "bad"}} {
		w := request("POST", "/api/services", tc.host, tc.origin, tc.token, nil)
		if w.Code != 403 {
			t.Fatalf("security status %d", w.Code)
		}
	}
	if w := request("GET", "/api/session", "127.0.0.1:4780", "https://evil.test", "", nil); w.Code != 403 {
		t.Fatal("cross origin session exposed")
	}
	projectDir := t.TempDir()
	in := input{Name: "test", Directory: "", Command: `cd "` + projectDir + `" powershell.exe -File .\\start.ps1`, Port: 3333}
	w := request("POST", "/api/services", "127.0.0.1:4780", "", s.token, in)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v model.Service
	json.Unmarshal(w.Body.Bytes(), &v)
	if v.Directory != projectDir || v.Command != `powershell.exe -File .\\start.ps1` {
		t.Fatalf("full launch command was not normalized: %+v", v)
	}
	in.Directory = projectDir
	in.Command = v.Command
	in.Name = "updated"
	w = request("PUT", "/api/services/"+v.ID, "127.0.0.1:4780", "", s.token, in)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var updated model.Service
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil || updated.Source != "manual" {
		t.Fatalf("editing service did not make it explicit: %s", w.Body.String())
	}
	w = request("GET", "/api/services", "127.0.0.1:4780", "", "", nil)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("updated")) {
		t.Fatal(w.Body.String())
	}
	w = request("DELETE", "/api/services/"+v.ID, "127.0.0.1:4780", "", s.token, nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	in.URL = "http://example.com:3333"
	w = request("POST", "/api/services", "127.0.0.1:4780", "", s.token, in)
	if w.Code != 400 {
		t.Fatal("remote URL accepted")
	}
}

func TestSettingsAPI(t *testing.T) {
	s, err := New(t.TempDir(), 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h := s.Handler(os.DirFS(t.TempDir()))
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "http://127.0.0.1:4780"+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-DevHub-Token", s.token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if w := request("GET", "/api/settings", nil); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"language":"en"`)) {
		t.Fatalf("default settings: %d %s", w.Code, w.Body.String())
	}
	w := request("PUT", "/api/settings", model.Settings{Language: "en", AutoScan: false, ScanIntervalSeconds: 30})
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"autoScan":false`)) {
		t.Fatalf("save settings: %d %s", w.Code, w.Body.String())
	}
	if w := request("PUT", "/api/settings", model.Settings{Language: "xx", AutoScan: true, ScanIntervalSeconds: 30}); w.Code != 400 {
		t.Fatalf("invalid language accepted: %d", w.Code)
	}
	if w := request("PUT", "/api/settings", model.Settings{Language: "en", AutoScan: true, ScanIntervalSeconds: 1}); w.Code != 400 {
		t.Fatalf("invalid interval accepted: %d", w.Code)
	}
	dir := t.TempDir()
	w = request("PUT", "/api/settings", model.Settings{
		Language:            "zh",
		AutoScan:            true,
		ScanIntervalSeconds: 10,
		IncludeDirectories:  []string{dir},
		ExcludeProcesses:    []string{"QQ.exe"},
		ExcludePorts:        []int{4301},
	})
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"QQ.exe"`)) || !bytes.Contains(w.Body.Bytes(), []byte("4301")) {
		t.Fatalf("discovery rules: %d %s", w.Code, w.Body.String())
	}
	if w := request("PUT", "/api/settings", model.Settings{
		Language:            "zh",
		AutoScan:            true,
		ScanIntervalSeconds: 10,
		ExcludeDirectories:  []string{"relative/path"},
	}); w.Code != 400 {
		t.Fatalf("relative discovery path accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestDiscoveryRulesPreserveManual(t *testing.T) {
	s, err := New(t.TempDir(), 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	dir := t.TempDir()
	discovered := model.Service{ID: "disc", Name: "auto", Project: "auto", Directory: dir, Port: 3911, Source: "discovered", ProcessName: "node"}
	manual := model.Service{ID: "man", Name: "kept", Project: "kept", Directory: dir, Port: 3912, Source: "manual", ProcessName: "node"}
	s.mu.Lock()
	s.services[discovered.ID] = discovered
	s.services[manual.ID] = manual
	s.mu.Unlock()
	if err := s.store.Save(discovered); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Save(manual); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.settings.ExcludeDirectories = []string{dir}
	s.mu.Unlock()
	s.applyDiscoveryFilter()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[discovered.ID]; ok {
		t.Fatal("excluded discovered service was kept")
	}
	if _, ok := s.services[manual.ID]; !ok {
		t.Fatal("manual service in excluded directory was deleted")
	}
}

func TestHealthProbeLayers(t *testing.T) {
	s, err := New(t.TempDir(), 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()
	httpURL := httpServer.URL
	port := mustPort(t, httpURL)
	healthy := s.health(model.Service{Port: port, URL: httpURL})
	if healthy.TCP != "reachable" || healthy.HTTP != "healthy" || healthy.HTTPStatus != 200 {
		t.Fatalf("healthy probe: %+v", healthy)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	broken := s.health(model.Service{Port: listener.Addr().(*net.TCPAddr).Port, URL: "http://127.0.0.1:" + fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)})
	if broken.TCP != "reachable" || broken.HTTP != "failed" {
		t.Fatalf("tcp-only probe: %+v", broken)
	}
	listener.Close()
	stopped := s.health(model.Service{Port: unusedPort(t), URL: "http://127.0.0.1:1"})
	if stopped.TCP != "unreachable" || stopped.HTTP != "unavailable" {
		t.Fatalf("stopped probe: %+v", stopped)
	}

	runningNoPort := s.health(model.Service{Managed: true, Port: unusedPort(t), URL: "http://127.0.0.1:1"})
	if runningNoPort.Process != "running" || runningNoPort.TCP != "unreachable" || runningNoPort.HTTP != "unavailable" || runningNoPort.Error != model.HealthProcessNoPort {
		t.Fatalf("process without port: %+v", runningNoPort)
	}
}

func TestStartupTimeoutFailure(t *testing.T) {
	s, err := New(t.TempDir(), 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	port := unusedPort(t)
	v := model.Service{
		ID:        "slow",
		Name:      "slow",
		Directory: t.TempDir(),
		Command:   "Start-Sleep -Seconds 60",
		Port:      port,
		URL:       fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	pid, birth, err := s.runner.Start(v)
	if err != nil {
		t.Fatal(err)
	}
	defer s.runner.Stop(v.ID)
	v.PID = pid
	v.Birth = birth
	s.mu.Lock()
	s.services[v.ID] = v
	s.startedAt[v.ID] = time.Now().Add(-StartupTimeout - time.Second)
	s.mu.Unlock()
	got := s.observe(v)
	if got.Status != "starting" || got.Failure != model.FailureStartTimeout {
		t.Fatalf("timeout failure: %+v", got)
	}
}

func mustPort(t *testing.T, raw string) int {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func unusedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	time.Sleep(20 * time.Millisecond)
	return port
}

func TestExportImportAndBackupAPI(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, 4780)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h := s.Handler(os.DirFS(t.TempDir()))
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "http://127.0.0.1:4780"+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-DevHub-Token", s.token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	project := t.TempDir()
	created := request("POST", "/api/services", map[string]any{
		"name": "api", "project": "Shop", "directory": project, "command": "npm run dev", "port": 3333, "url": "http://127.0.0.1:3333",
		"env": []map[string]any{{"name": "API_TOKEN", "value": "hidden-value", "secret": true}},
	})
	if created.Code != 200 {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	exported := request("POST", "/api/export", map[string]bool{"logs": false, "secrets": false})
	if exported.Code != 200 || bytes.Contains(exported.Body.Bytes(), []byte("hidden-value")) {
		t.Fatalf("export leaked secrets: %d %s", exported.Code, exported.Body.String())
	}
	if w := request("POST", "/api/backup", map[string]any{}); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(".db")) {
		t.Fatalf("backup: %d %s", w.Code, w.Body.String())
	}
	empty := t.TempDir()
	other, err := New(empty, 4781)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	oh := other.Handler(os.DirFS(t.TempDir()))
	req := httptest.NewRequest("POST", "http://127.0.0.1:4781/api/import", bytes.NewReader(exported.Body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DevHub-Token", other.token)
	w := httptest.NewRecorder()
	oh.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("import: %d %s", w.Code, w.Body.String())
	}
	listed := httptest.NewRecorder()
	oh.ServeHTTP(listed, httptest.NewRequest("GET", "http://127.0.0.1:4781/api/services", nil))
	if listed.Code != 200 || !bytes.Contains(listed.Body.Bytes(), []byte(`"name":"api"`)) {
		t.Fatalf("imported services: %d %s", listed.Code, listed.Body.String())
	}
	bad := request("POST", "/api/import", map[string]any{"app": "nope", "format": 1, "settings": model.DefaultSettings(), "services": []any{}})
	if bad.Code != 400 {
		t.Fatalf("bad import: %d %s", bad.Code, bad.Body.String())
	}
}
