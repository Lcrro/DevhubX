package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Lcrro/DevhubX/internal/model"
)

func TestDecideIncludeExclude(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "app")
	other := filepath.Join(t.TempDir(), "other")
	filter := Filter{
		IncludeDirectories: []string{root},
		ExcludeDirectories: []string{filepath.Join(root, "skip")},
		ExcludeProcesses:   []string{"QQ.exe"},
		ExcludePorts:       []int{4301},
	}
	if got := Decide(inside, "node.exe", 3000, filter); !got.Keep || got.Reason != ReasonIncludeDirectory {
		t.Fatalf("include directory: %+v", got)
	}
	if got := Decide(other, "node.exe", 3000, filter); got.Keep || got.Reason != ReasonIncludeDirMiss {
		t.Fatalf("outside include directory: %+v", got)
	}
	if got := Decide(filepath.Join(root, "skip", "svc"), "node.exe", 3000, filter); got.Keep || got.Reason != ReasonExcludeDirectory {
		t.Fatalf("exclude directory: %+v", got)
	}
	if got := Decide(inside, "QQ.exe", 3000, filter); got.Keep || got.Reason != ReasonExcludeProcess {
		t.Fatalf("exclude process: %+v", got)
	}
	if got := Decide(inside, "node", 4301, filter); got.Keep || got.Reason != ReasonExcludePort {
		t.Fatalf("exclude port: %+v", got)
	}
}

func TestDecideSystemInstallStillWins(t *testing.T) {
	var managed string
	switch runtime.GOOS {
	case "windows":
		managed = filepath.Join(os.Getenv("ProgramFiles"), "Tencent", "QQNT")
	case "darwin":
		managed = "/Applications/QQ.app/Contents/Resources/app"
	default:
		managed = "/opt/qq/resources/app"
	}
	got := Decide(managed, "node", 3000, Filter{IncludeDirectories: []string{managed}})
	if got.Keep || got.Reason != ReasonSystemInstall {
		t.Fatalf("system install should stay excluded: %+v", got)
	}
}

func TestMergeByPIDAndPrimaryPort(t *testing.T) {
	items := []model.Service{
		{PID: 11, Port: 9229, URL: "http://127.0.0.1:9229", DiscoveryReason: ReasonProjectMarker, DiscoveryDetail: "vite"},
		{PID: 11, Port: 5173, URL: "http://127.0.0.1:5173", DiscoveryReason: ReasonProjectMarker, DiscoveryDetail: "vite"},
		{PID: 12, Port: 8080, URL: "http://127.0.0.1:8080", DiscoveryReason: ReasonProjectMarker},
	}
	got := MergeByPID(items)
	if len(got) != 2 {
		t.Fatalf("merged count: %d", len(got))
	}
	var merged model.Service
	for _, item := range got {
		if item.PID == 11 {
			merged = item
		}
	}
	if merged.Port != 5173 || len(merged.Ports) != 2 || merged.DiscoveryReason != ReasonMergedPorts {
		t.Fatalf("primary/merge: %+v", merged)
	}
	if merged.URL != "http://127.0.0.1:5173" {
		t.Fatalf("url rewritten: %s", merged.URL)
	}
	if ChoosePrimaryPort([]int{9229, 3000, 5173}, 9229) != 9229 {
		t.Fatal("preferred port was not kept")
	}
}

func TestServiceExcludedKeepsManual(t *testing.T) {
	dir := t.TempDir()
	manual := model.Service{Source: "manual", Directory: dir, Port: 3000, ProcessName: "node"}
	discovered := model.Service{Source: "discovered", Directory: dir, Port: 3000, ProcessName: "node"}
	filter := Filter{ExcludeDirectories: []string{dir}}
	if ServiceExcluded(manual, filter) {
		t.Fatal("manual service was excluded")
	}
	if !ServiceExcluded(discovered, filter) {
		t.Fatal("discovered service in excluded directory was kept")
	}
}

func TestProcessNameNormalization(t *testing.T) {
	if !matchProcess("node.exe", []string{"NODE"}) {
		t.Fatal("process name should match without extension and case")
	}
	if !matchProcess(filepath.Join("usr", "bin", "node"), []string{"node"}) {
		t.Fatal("unix process path was not matched by basename")
	}
	if matchProcess("python", []string{"node"}) {
		t.Fatal("unrelated process matched")
	}
}
