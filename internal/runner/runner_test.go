package runner

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"devhub/internal/model"
	"devhub/internal/store"
)

func TestHelperServer(t *testing.T) {
	if os.Getenv("DEVHUB_HELPER") != "1" {
		return
	}
	fmt.Println("fixture service ready")
	if err := http.ListenAndServe(os.Getenv("DEVHUB_HELPER_ADDR"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("DevHub fixture")) })); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestIPv6PortConflict(t *testing.T) {
	l, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skip("IPv6 loopback unavailable")
	}
	defer l.Close()
	if !PortOpen(l.Addr().(*net.TCPAddr).Port) {
		t.Fatal("IPv6-only listener was missed")
	}
}
func TestLifecycleAndPortConflict(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	t.Setenv("DEVHUB_HELPER", "1")
	t.Setenv("DEVHUB_HELPER_ADDR", fmt.Sprintf("127.0.0.1:%d", port))
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	line := "'" + strings.ReplaceAll(exe, "'", "'\"'\"'") + "' -test.run=^TestHelperServer$"
	if runtime.GOOS == "windows" {
		line = "& '" + strings.ReplaceAll(exe, "'", "''") + "' '-test.run=^TestHelperServer$'"
	}
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m := New(s)
	defer m.Close()
	v := model.Service{ID: "fixture", Directory: t.TempDir(), Command: line, Port: port}
	pid, _, err := m.Start(v)
	if err != nil || pid == 0 {
		t.Fatal("start", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for !PortOpen(port) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if !PortOpen(port) {
		logs, _ := s.Logs(v.ID, 0)
		t.Fatalf("server never became ready: %+v", logs)
	}
	v.ID = "conflict"
	if _, _, err = m.Start(v); err == nil {
		t.Fatal("port conflict accepted")
	}
	if err = m.Stop("fixture"); err != nil {
		t.Fatal(err)
	}
	if PortOpen(port) {
		t.Fatal("child listener survived stop")
	}
	logs, err := s.Logs("fixture", 0)
	if err != nil || len(logs) == 0 {
		t.Fatal("logs missing", err)
	}
}

func TestQuickExitIsRecorded(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "exit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m := New(s)
	defer m.Close()
	v := model.Service{ID: "exit", Directory: t.TempDir(), Command: "exit 7", Port: unusedListenPort(t)}
	if _, _, err := m.Start(v); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	var msg string
	for time.Now().Before(deadline) {
		msg = m.ConsumeExit(v.ID)
		if msg != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if msg == "" {
		t.Fatal("quick exit was not recorded")
	}
}

func unusedListenPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
