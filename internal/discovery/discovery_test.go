package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Lcrro/DevhubX/internal/model"
)

func TestIdentifyNestedProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo","devDependencies":{"vite":"1"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "src", "app")
	os.MkdirAll(dir, 0700)
	name, path, kind := Identify(dir)
	if name != "demo" || path != root || kind != "vite" {
		t.Fatalf("%s %s %s", name, path, kind)
	}
}

func TestDiscoveryHelper(t *testing.T) {
	if os.Getenv("DEVHUB_DISCOVERY_HELPER") != "1" {
		return
	}
	addrs := []string{os.Getenv("DEVHUB_DISCOVERY_ADDR")}
	if extra := os.Getenv("DEVHUB_DISCOVERY_ADDRS"); extra != "" {
		addrs = strings.Split(extra, ",")
	}
	errc := make(chan error, 1)
	started := 0
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		started++
		go func(a string) {
			errc <- http.ListenAndServe(a, http.NewServeMux())
		}(addr)
	}
	if started == 0 {
		os.Exit(2)
	}
	if err := <-errc; err != nil {
		os.Exit(2)
	}
}

func TestDiscoverRealListener(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"discovery-fixture","dependencies":{"react":"19"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	c := exec.Command(os.Args[0], "-test.run=^TestDiscoveryHelper$")
	c.Dir = dir
	c.Env = append(os.Environ(), "DEVHUB_DISCOVERY_HELPER=1", fmt.Sprintf("DEVHUB_DISCOVERY_ADDR=127.0.0.1:%d", port))
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { c.Process.Kill(); c.Wait() }()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, e := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if e == nil {
			conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	found, err := Scan(ctx, 4780, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range found {
		if v.PID == int32(c.Process.Pid) && v.Port == port {
			if v.Project != "discovery-fixture" || v.Framework != "react" {
				t.Fatalf("incorrect identity: %+v", v)
			}
			if v.DiscoveryReason == "" || v.ProcessName == "" {
				t.Fatalf("missing discovery explanation: %+v", v)
			}
			return
		}
	}
	t.Fatal("current-user development listener was not discovered")
}

func TestDiscoverMergeAndExcludePort(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"multi-port","devDependencies":{"vite":"1"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	second, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	portA := first.Addr().(*net.TCPAddr).Port
	portB := second.Addr().(*net.TCPAddr).Port
	first.Close()
	second.Close()
	c := exec.Command(os.Args[0], "-test.run=^TestDiscoveryHelper$")
	c.Dir = dir
	c.Env = append(os.Environ(), "DEVHUB_DISCOVERY_HELPER=1", fmt.Sprintf("DEVHUB_DISCOVERY_ADDRS=127.0.0.1:%d,127.0.0.1:%d", portA, portB))
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { c.Process.Kill(); c.Wait() }()
	waitForPort(t, portA)
	waitForPort(t, portB)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	found, err := Scan(ctx, 4780, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	var merged *model.Service
	for i := range found {
		if found[i].PID == int32(c.Process.Pid) {
			merged = &found[i]
			break
		}
	}
	if merged == nil || len(merged.Ports) != 2 {
		t.Fatalf("same process ports were not merged: %+v", merged)
	}
	excluded, err := Scan(ctx, 4780, Filter{ExcludePorts: []int{portB}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range excluded {
		if v.PID == int32(c.Process.Pid) {
			if containsPort(v.Ports, portB) || v.Port == portB {
				t.Fatalf("excluded port still present: %+v", v)
			}
			return
		}
	}
	t.Fatal("remaining included port was not discovered")
}

func waitForPort(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("port %d did not open", port)
}
func TestUnknownDirectory(t *testing.T) {
	_, _, kind := Identify(t.TempDir())
	if kind != "" {
		t.Fatal(kind)
	}
}

func TestPathWithin(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "vendor", "desktop-app")
	outside := root + "-other"
	if !pathWithin(root, inside) {
		t.Fatal("nested application directory was not recognized")
	}
	if pathWithin(root, outside) {
		t.Fatal("sibling directory with a shared prefix was treated as nested")
	}
}

func TestManagedInstallPath(t *testing.T) {
	var managed, project string
	switch runtime.GOOS {
	case "windows":
		managed = filepath.Join(os.Getenv("ProgramFiles"), "Tencent", "QQNT", "resources", "app")
		project = filepath.Join(os.Getenv("USERPROFILE"), "Projects", "qq-chat")
	case "darwin":
		managed = "/Applications/QQ.app/Contents/Resources/app"
		project = filepath.Join(os.Getenv("HOME"), "Projects", "qq-chat")
	default:
		managed = "/opt/qq/resources/app"
		project = filepath.Join(os.Getenv("HOME"), "Projects", "qq-chat")
	}
	if !ManagedInstallPath(managed) {
		t.Fatal("managed application directory was not excluded")
	}
	if ManagedInstallPath(project) {
		t.Fatal("user project directory was excluded")
	}
}
