package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	"devhub/internal/cover"
	"devhub/internal/discovery"
	"devhub/internal/launchcmd"
	"devhub/internal/model"
	"devhub/internal/runner"
	"devhub/internal/store"
)

type Server struct {
	mu         sync.Mutex
	store      *store.Store
	runner     *runner.Manager
	services   map[string]model.Service
	data       string
	port       int
	token      string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	scanMu     sync.Mutex
	coverQueue chan string
	pending    map[string]bool
	attempted  map[string]time.Time
	lastScan   string
	scanError  string
	settings   model.Settings
	scanWake   chan struct{}
	startedAt  map[string]time.Time
}

func New(data string, port int) (*Server, error) {
	if err := os.MkdirAll(filepath.Join(data, "covers"), 0700); err != nil {
		return nil, err
	}
	db, err := store.Open(filepath.Join(data, "devhub.db"))
	if err != nil {
		return nil, err
	}
	all, err := db.List()
	if err != nil {
		db.Close()
		return nil, err
	}
	settings, err := db.LoadSettings()
	if err != nil {
		db.Close()
		return nil, err
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		db.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{store: db, runner: runner.New(db), services: map[string]model.Service{}, data: data, port: port, token: hex.EncodeToString(b), ctx: ctx, cancel: cancel, coverQueue: make(chan string, 64), pending: map[string]bool{}, attempted: map[string]time.Time{}, settings: settings, scanWake: make(chan struct{}, 1), startedAt: map[string]time.Time{}}
	for _, v := range all {
		v.Managed = false
		s.services[v.ID] = v
	}
	return s, nil
}
func (s *Server) Start() {
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.scanLoop()
	}()
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case id := <-s.coverQueue:
				s.capture(id)
			}
		}
	}()
}

func (s *Server) getSettings() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *Server) scanLoop() {
	first := true
	for {
		settings := s.getSettings()
		if !settings.AutoScan {
			select {
			case <-s.ctx.Done():
				return
			case <-s.scanWake:
				continue
			}
		}
		if first {
			s.Scan()
			first = false
			continue
		}
		timer := time.NewTimer(time.Duration(settings.ScanIntervalSeconds) * time.Second)
		select {
		case <-s.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-s.scanWake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			continue
		case <-timer.C:
			s.Scan()
		}
	}
}
func (s *Server) Close() { s.cancel(); s.wg.Wait(); s.runner.Close(); _ = s.store.Close() }
func ID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func alive(v model.Service) bool {
	if v.PID <= 0 || v.Birth == 0 {
		return false
	}
	p, err := process.NewProcess(v.PID)
	if err != nil {
		return false
	}
	birth, err := p.CreateTime()
	return err == nil && birth == v.Birth
}
func (s *Server) status(v model.Service) model.Service {
	v.Managed = s.runner.Has(v.ID)
	if v.Managed {
		if runner.PortOpen(v.Port) {
			v.Status = "running"
		} else {
			v.Status = "starting"
		}
	} else if alive(v) {
		v.Status = "external"
	} else {
		v.Status = "stopped"
		v.PID = 0
	}
	return v
}

const StartupTimeout = 20 * time.Second

func (s *Server) health(v model.Service) model.Health {
	h := model.Health{Process: "stopped", TCP: "unreachable", HTTP: "unavailable", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	if v.Managed || alive(v) {
		h.Process = "running"
	}
	if !runner.PortOpen(v.Port) {
		if h.Process == "running" {
			h.Error = model.HealthProcessNoPort
		}
		return h
	}
	h.TCP = "reachable"
	ctx, cancel := context.WithTimeout(s.ctx, 1200*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.URL, nil)
	if err != nil {
		h.HTTP = "failed"
		h.Error = err.Error()
		return h
	}
	client := &http.Client{
		Timeout: 1200 * time.Millisecond,
		CheckRedirect: func(next *http.Request, _ []*http.Request) error {
			if !cover.LocalURL(next.URL.String()) {
				return errors.New("HTTP 重定向到非本机地址")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		h.HTTP = "failed"
		h.Error = err.Error()
		return h
	}
	defer resp.Body.Close()
	h.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		h.HTTP = "healthy"
		return h
	}
	h.HTTP = "failed"
	h.Error = resp.Status
	return h
}

func (s *Server) removeServiceLocked(id string, v model.Service) {
	if err := s.store.Delete(id); err != nil {
		log.Printf("remove discovered service %s: %v", id, err)
		return
	}
	delete(s.services, id)
	delete(s.pending, id)
	delete(s.attempted, id)
	if v.Cover != "" {
		_ = os.Remove(filepath.Join(s.data, "covers", v.Cover))
	}
}

func sameDirectory(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func portsOf(v model.Service) []int {
	if len(v.Ports) > 0 {
		return v.Ports
	}
	if v.Port > 0 {
		return []int{v.Port}
	}
	return nil
}

func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func portOverlap(old, found model.Service) bool {
	if old.PID > 0 && found.PID > 0 && old.PID == found.PID {
		return true
	}
	for _, a := range portsOf(old) {
		if containsInt(portsOf(found), a) {
			return true
		}
	}
	return old.Port == found.Port
}

func settingsView(settings model.Settings) map[string]any {
	settings = settings.Normalized()
	return map[string]any{
		"language":               settings.Language,
		"autoScan":               settings.AutoScan,
		"scanIntervalSeconds":    settings.ScanIntervalSeconds,
		"minScanIntervalSeconds": model.MinScanIntervalSeconds,
		"maxScanIntervalSeconds": model.MaxScanIntervalSeconds,
		"includeDirectories":     settings.IncludeDirectories,
		"excludeDirectories":     settings.ExcludeDirectories,
		"includeProcesses":       settings.IncludeProcesses,
		"excludeProcesses":       settings.ExcludeProcesses,
		"includePorts":           settings.IncludePorts,
		"excludePorts":           settings.ExcludePorts,
		"theme":                  settings.Theme,
		"accent":                 settings.Accent,
		"density":                settings.Density,
		"serviceView":            settings.ServiceView,
		"projectPrefs":           settings.ProjectPrefs,
	}
}

func (s *Server) Scan() {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	s.mu.Lock()
	filter := discovery.FilterFromSettings(s.settings)
	s.mu.Unlock()
	found, err := discovery.Scan(ctx, s.port, filter)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastScan = time.Now().Format(time.RFC3339)
	s.scanError = ""
	if err != nil {
		s.scanError = err.Error()
	}
	if err == nil {
		for id, v := range s.services {
			if discovery.ServiceExcluded(v, filter) {
				s.removeServiceLocked(id, v)
			}
		}
	}
	for _, v := range found {
		matched := false
		bestID := ""
		bestRank := -1
		for id, old := range s.services {
			if !portOverlap(old, v) {
				continue
			}
			rank := -1
			switch {
			case s.runner.Has(id):
				rank = 3
			case sameDirectory(old.Directory, v.Directory) && old.Source == "manual":
				rank = 2
			case sameDirectory(old.Directory, v.Directory) && old.Source == "discovered":
				rank = 1
			}
			if rank > bestRank {
				bestID, bestRank = id, rank
			}
		}
		if bestID != "" {
			matched = true
			old := s.services[bestID]
			old.ProcessName = v.ProcessName
			old.Ports = v.Ports
			old.DiscoveryReason = v.DiscoveryReason
			old.DiscoveryDetail = v.DiscoveryDetail
			if !s.runner.Has(bestID) {
				old.PID = v.PID
				old.Birth = v.Birth
			}
			if !containsInt(portsOf(v), old.Port) {
				old.URL = discovery.URLForPort(old.URL, v.Port)
				old.Port = v.Port
			}
			s.services[bestID] = old
			if e := s.store.Save(old); e != nil {
				log.Print(e)
			}
			if bestRank >= 1 {
				for id, duplicate := range s.services {
					if id == bestID || duplicate.Source != "discovered" {
						continue
					}
					samePID := v.PID > 0 && duplicate.PID == v.PID
					samePortDir := portOverlap(duplicate, v) && sameDirectory(duplicate.Directory, v.Directory)
					if samePID || samePortDir {
						s.removeServiceLocked(id, duplicate)
					}
				}
			}
		}
		if !matched {
			v.ID = ID()
			if e := s.store.Save(v); e == nil {
				s.services[v.ID] = v
			} else {
				log.Print(e)
			}
		}
	}
	for id, v := range s.services {
		v = s.status(v)
		s.services[id] = v
		if (v.Status == "running" || v.Status == "external") && v.Cover == "" && time.Since(s.attempted[id]) > 5*time.Minute {
			s.queueLocked(id)
		}
	}
}
func (s *Server) queueLocked(id string) {
	if s.pending[id] {
		return
	}
	select {
	case s.coverQueue <- id:
		s.pending[id] = true
		s.attempted[id] = time.Now()
	default:
	}
}
func (s *Server) capture(id string) {
	s.mu.Lock()
	v, ok := s.services[id]
	s.mu.Unlock()
	if !ok {
		return
	}
	file := id + "-" + ID() + ".png"
	err := cover.Capture(s.ctx, v.URL, filepath.Join(s.data, "covers", file))
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
	current, ok := s.services[id]
	if !ok {
		_ = os.Remove(filepath.Join(s.data, "covers", file))
		return
	}
	if err != nil {
		current.ScreenshotError = err.Error()
	} else {
		old := current.Cover
		current.Cover = file
		current.ScreenshotError = ""
		if old != "" {
			_ = os.Remove(filepath.Join(s.data, "covers", old))
		}
	}
	if e := s.store.Save(current); e != nil {
		log.Print(e)
	}
	s.services[id] = current
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, err error) {
	write(w, status, map[string]string{"error": err.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	return decodeLimit(w, r, v, 64<<10)
}

func decodeLimit(w http.ResponseWriter, r *http.Request, v any, max int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("请求只能包含一个 JSON 对象")
	}
	return nil
}

func (s *Server) Handler(assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, map[string]any{"token": s.token, "platform": platform(), "browserAvailable": cover.Browser() != "", "dataDirectory": s.data})
	})
	mux.HandleFunc("GET /api/services", s.list)
	mux.HandleFunc("GET /api/settings", s.getSettingsAPI)
	mux.HandleFunc("PUT /api/settings", s.updateSettings)
	mux.HandleFunc("POST /api/export", s.exportConfig)
	mux.HandleFunc("POST /api/import", s.importConfig)
	mux.HandleFunc("POST /api/backup", s.backupConfig)
	mux.HandleFunc("POST /api/scan", func(w http.ResponseWriter, r *http.Request) { s.Scan(); s.list(w, r) })
	mux.HandleFunc("POST /api/services", s.save)
	mux.HandleFunc("PUT /api/services/{id}", s.save)
	mux.HandleFunc("DELETE /api/services/{id}", s.remove)
	mux.HandleFunc("POST /api/services/{id}/{action}", s.action)
	mux.HandleFunc("GET /api/services/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		result, err := s.store.Logs(r.PathValue("id"), after)
		if err != nil {
			fail(w, 500, err)
			return
		}
		write(w, 200, result)
	})
	mux.HandleFunc("DELETE /api/services/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.ClearLogs(r.PathValue("id")); err != nil {
			fail(w, 500, err)
			return
		}
		write(w, 200, map[string]bool{"ok": true})
	})
	mux.HandleFunc("GET /covers/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		s.mu.Lock()
		allowed := false
		for _, v := range s.services {
			if v.Cover != "" && v.Cover == name {
				allowed = true
				break
			}
		}
		s.mu.Unlock()
		if !allowed {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.data, "covers", name))
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, errors.New("API 不存在")) })
	files := http.FileServer(http.FS(assets))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/" {
			if _, err := fs.Stat(assets, "index.html"); err != nil {
				http.Error(w, "Frontend missing. Run npm ci && npm run build in web, then rebuild DevHub.", 503)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
	return s.security(mux)
}

func (s *Server) getSettingsAPI(w http.ResponseWriter, r *http.Request) {
	write(w, 200, settingsView(s.getSettings()))
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var in model.Settings
	if err := decode(w, r, &in); err != nil {
		fail(w, 400, err)
		return
	}
	if err := in.Validate(); err != nil {
		fail(w, 400, err)
		return
	}
	in = in.Normalized()
	s.mu.Lock()
	rulesChanged := !discovery.FilterFromSettings(s.settings).Equal(discovery.FilterFromSettings(in))
	s.settings = in
	err := s.store.SaveSettings(in)
	autoScan := in.AutoScan
	if err == nil && rulesChanged {
		s.wg.Add(1)
	}
	s.mu.Unlock()
	if err != nil {
		fail(w, 500, err)
		return
	}
	select {
	case s.scanWake <- struct{}{}:
	default:
	}
	if rulesChanged {
		go func() {
			defer s.wg.Done()
			if autoScan {
				s.Scan()
			} else {
				s.applyDiscoveryFilter()
			}
		}()
	}
	write(w, 200, settingsView(in))
}

func (s *Server) exportConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Logs    bool `json:"logs"`
		Secrets bool `json:"secrets"`
	}
	if err := decode(w, r, &req); err != nil {
		fail(w, 400, err)
		return
	}
	opts := store.ExportOptions{IncludeLogs: req.Logs, IncludeSecrets: req.Secrets}
	bundle, err := s.store.Export(opts)
	if err != nil {
		fail(w, 500, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="devhub-export.json"`)
	write(w, 200, bundle)
}

func (s *Server) importConfig(w http.ResponseWriter, r *http.Request) {
	var bundle store.Bundle
	if err := decodeLimit(w, r, &bundle, 4<<20); err != nil {
		fail(w, 400, err)
		return
	}
	if err := store.ValidateBundle(bundle); err != nil {
		fail(w, 400, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.services {
		if s.runner.Has(v.ID) {
			fail(w, 409, errors.New("请先停止由 DevHub 启动的服务再导入"))
			return
		}
	}
	backup := filepath.Join(s.data, "backups", time.Now().UTC().Format("devhub-20060102-150405.000")+"-"+ID()[:8]+".db")
	if err := s.store.Backup(backup); err != nil {
		fail(w, 500, fmt.Errorf("导入前备份失败，已中止: %w", err))
		return
	}
	services := make([]model.Service, 0, len(bundle.Services))
	for _, v := range bundle.Services {
		services = append(services, store.SanitizeImported(v))
	}
	if err := s.store.ReplaceAll(bundle.Settings.Normalized(), services, bundle.Logs); err != nil {
		fail(w, 500, fmt.Errorf("导入失败，原数据库未改写: %w", err))
		return
	}
	all, err := s.store.List()
	if err != nil {
		fail(w, 500, err)
		return
	}
	next := map[string]model.Service{}
	for _, v := range all {
		next[v.ID] = v
	}
	s.services = next
	s.settings = bundle.Settings.Normalized()
	write(w, 200, map[string]any{"ok": true, "backup": backup, "imported": len(services)})
}

func (s *Server) backupConfig(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.data, "backups", time.Now().UTC().Format("devhub-20060102-150405.000")+"-"+ID()[:8]+".db")
	if err := s.store.Backup(path); err != nil {
		fail(w, 500, err)
		return
	}
	write(w, 200, map[string]any{"ok": true, "path": path})
}

func (s *Server) applyDiscoveryFilter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	filter := discovery.FilterFromSettings(s.settings)
	for id, v := range s.services {
		if discovery.ServiceExcluded(v, filter) {
			s.removeServiceLocked(id, v)
		}
	}
}
func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		host := fmt.Sprintf("127.0.0.1:%d", s.port)
		localhost := fmt.Sprintf("localhost:%d", s.port)
		if r.Host != host && r.Host != localhost {
			fail(w, 403, errors.New("不允许的 Host"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			fail(w, 403, errors.New("不允许跨站请求"))
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" {
			fail(w, 403, errors.New("不允许跨站请求"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != "GET" {
			if r.Header.Get("X-DevHub-Token") != s.token {
				fail(w, 403, errors.New("会话已更新，请刷新页面"))
				return
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				fail(w, 415, errors.New("需要 application/json"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) observe(v model.Service) model.Service {
	v = s.status(v)
	v.Health = s.health(v)
	if msg := s.runner.ConsumeExit(v.ID); msg != "" {
		v.Failure = model.FailureStartExit
		v.FailureDetail = msg
		return v
	}
	if v.Status == "starting" {
		s.mu.Lock()
		started, ok := s.startedAt[v.ID]
		s.mu.Unlock()
		if ok && time.Since(started) >= StartupTimeout {
			v.Failure = model.FailureStartTimeout
			v.FailureDetail = ""
		}
	}
	if v.Status == "running" && v.Failure == model.FailureStartTimeout {
		v.Failure = ""
		v.FailureDetail = ""
	}
	return v
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	items := make([]model.Service, 0, len(s.services))
	for _, v := range s.services {
		items = append(items, v)
	}
	lastScan, scanError := s.lastScan, s.scanError
	s.mu.Unlock()
	all := []model.Service{}
	for _, v := range items {
		v = s.observe(v)
		all = append(all, v)
		s.mu.Lock()
		if current, ok := s.services[v.ID]; ok {
			changed := current.Failure != v.Failure || current.FailureDetail != v.FailureDetail || current.Status != v.Status || current.PID != v.PID || current.Managed != v.Managed
			current.Health = v.Health
			current.Failure = v.Failure
			current.FailureDetail = v.FailureDetail
			current.Status = v.Status
			current.PID = v.PID
			current.Birth = v.Birth
			current.Managed = v.Managed
			s.services[v.ID] = current
			if changed {
				if e := s.store.Save(current); e != nil {
					log.Print(e)
				}
			}
		}
		s.mu.Unlock()
	}
	write(w, 200, map[string]any{"services": all, "lastScan": lastScan, "scanError": scanError})
}

type input struct {
	Name      string         `json:"name"`
	Project   string         `json:"project"`
	Directory string         `json:"directory"`
	Command   string         `json:"command"`
	Port      int            `json:"port"`
	URL       string         `json:"url"`
	Env       []model.EnvVar `json:"env"`
}

func validate(in *input) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Project = strings.TrimSpace(in.Project)
	in.Directory = strings.TrimSpace(in.Directory)
	in.Command = strings.TrimSpace(in.Command)
	if in.Name == "" || len(in.Name) > 100 || len(in.Project) > 100 {
		return errors.New("请输入不超过 100 字节的服务名称和项目名称")
	}
	if !filepath.IsAbs(in.Directory) {
		return errors.New("项目目录必须是绝对路径")
	}
	info, err := os.Stat(in.Directory)
	if err != nil || !info.IsDir() {
		return errors.New("项目目录不存在")
	}
	if in.Port < 1 || in.Port > 65535 {
		return errors.New("端口必须在 1–65535 之间")
	}
	if len(in.Command) > 8192 {
		return errors.New("启动命令过长")
	}
	in.Env = model.NormalizeEnv(in.Env)
	if in.URL == "" {
		in.URL = fmt.Sprintf("http://127.0.0.1:%d", in.Port)
	}
	if !cover.LocalURL(in.URL) {
		return errors.New("网页地址必须是本机 HTTP/HTTPS URL")
	}
	u, _ := url.Parse(in.URL)
	p := u.Port()
	if p == "" {
		if u.Scheme == "https" {
			p = "443"
		} else {
			p = "80"
		}
	}
	if p != strconv.Itoa(in.Port) {
		return errors.New("网页地址端口必须与服务端口一致")
	}
	return nil
}
func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	var in input
	if err := decode(w, r, &in); err != nil {
		fail(w, 400, err)
		return
	}
	in.Directory, in.Command, _ = launchcmd.Normalize(in.Directory, in.Command)
	if err := validate(&in); err != nil {
		fail(w, 400, err)
		return
	}
	if in.Port == s.port {
		fail(w, 400, errors.New("不能使用 DevHub 自身端口"))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := r.PathValue("id")
	v := model.Service{ID: ID(), Source: "manual", Status: "stopped"}
	if id != "" {
		var ok bool
		v, ok = s.services[id]
		if !ok {
			fail(w, 404, errors.New("服务不存在"))
			return
		}
		if s.status(v).Status != "stopped" {
			fail(w, 409, errors.New("请先停止服务再编辑"))
			return
		}
		// Editing an auto-discovered service turns it into an explicit user
		// configuration, so future scans prefer it over duplicate observations.
		v.Source = "manual"
	}
	name, _, kind := discovery.Identify(in.Directory)
	if in.Project == "" {
		in.Project = name
	}
	if v.Port != in.Port || v.URL != in.URL {
		v.Cover = ""
		v.ScreenshotError = ""
	}
	v.Name = in.Name
	v.Project = in.Project
	v.Directory = filepath.Clean(in.Directory)
	v.Command = in.Command
	v.Port = in.Port
	v.URL = in.URL
	v.Env = in.Env
	v.Framework = kind
	if err := s.store.Save(v); err != nil {
		fail(w, 500, err)
		return
	}
	s.services[v.ID] = v
	write(w, 200, v)
}
func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := r.PathValue("id")
	v, ok := s.services[id]
	if !ok {
		fail(w, 404, errors.New("服务不存在"))
		return
	}
	if s.status(v).Status != "stopped" {
		fail(w, 409, errors.New("请先停止服务再移除"))
		return
	}
	if err := s.store.Delete(id); err != nil {
		fail(w, 500, err)
		return
	}
	delete(s.services, id)
	delete(s.pending, id)
	delete(s.attempted, id)
	if v.Cover != "" {
		_ = os.Remove(filepath.Join(s.data, "covers", v.Cover))
	}
	write(w, 200, map[string]bool{"ok": true})
}
func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := r.PathValue("id")
	v, ok := s.services[id]
	if !ok {
		fail(w, 404, errors.New("服务不存在"))
		return
	}
	v = s.status(v)
	switch r.PathValue("action") {
	case "start":
		if v.Status != "stopped" {
			fail(w, 409, errors.New("服务已运行"))
			return
		}
		pid, birth, err := s.runner.Start(v)
		if err != nil {
			v.Failure = model.FailureStartExit
			v.FailureDetail = err.Error()
			s.services[id] = v
			_ = s.store.Save(v)
			fail(w, 409, err)
			return
		}
		v.PID = pid
		v.Birth = birth
		v.Managed = true
		v.Status = "starting"
		v.Failure = ""
		v.FailureDetail = ""
		s.startedAt[id] = time.Now()
	case "stop":
		if v.Managed {
			if err := s.runner.Stop(id); err != nil {
				fail(w, 409, err)
				return
			}
		} else {
			var req struct {
				Confirm bool `json:"confirm"`
			}
			if err := decode(w, r, &req); err != nil || !req.Confirm {
				fail(w, 400, errors.New("停止外部服务需要明确确认"))
				return
			}
			p, err := process.NewProcess(v.PID)
			if err != nil || !alive(v) || !discovery.Owned(p) || v.PID == int32(os.Getpid()) {
				fail(w, 409, errors.New("进程身份已改变或无权停止，请刷新"))
				return
			}
			// External processes are not in our process group: terminate only the verified PID.
			if err = p.Kill(); err != nil {
				fail(w, 409, err)
				return
			}
			_ = s.store.Append(id, "[DevHub] 已请求停止外部进程；外部终端的历史日志不可读取。\n")
		}
		v.Status = "stopped"
		v.Managed = false
		v.PID = 0
		v.Birth = 0
		if v.Failure == model.FailureStartTimeout {
			v.Failure = ""
			v.FailureDetail = ""
		}
		delete(s.startedAt, id)
	case "capture":
		if v.Status == "stopped" {
			fail(w, 409, errors.New("请先启动服务"))
			return
		}
		s.queueLocked(id)
		write(w, 202, map[string]bool{"queued": true})
		return
	default:
		fail(w, 404, errors.New("未知操作"))
		return
	}
	s.services[id] = v
	if err := s.store.Save(v); err != nil {
		fail(w, 500, err)
		return
	}
	write(w, 200, v)
}

func ListenAddress(address string) (int, error) {
	host, p, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}
	if host != "127.0.0.1" {
		return 0, errors.New("安全起见，只允许监听 127.0.0.1")
	}
	port, err := strconv.Atoi(p)
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("无效端口")
	}
	return port, nil
}
