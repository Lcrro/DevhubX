package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"devhub/internal/model"
)

// ManagedInstallPath reports directories owned by the operating system's app
// installation machinery. Desktop apps often ship package.json/go.mod files
// and open loopback ports internally; those are not development projects.
func ManagedInstallPath(dir string) bool {
	if dir == "" {
		return false
	}
	roots := []string{}
	switch runtime.GOOS {
	case "windows":
		roots = []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData"), os.Getenv("SystemRoot")}
	case "darwin":
		roots = []string{"/Applications", "/System", "/Library"}
	default:
		roots = []string{"/usr", "/opt", "/snap", "/var/lib/flatpak"}
	}
	for _, root := range roots {
		if pathWithin(root, dir) {
			return true
		}
	}
	return false
}

func pathWithin(root, candidate string) bool {
	if root == "" || candidate == "" {
		return false
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	candidate, err = filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Identify walks up at most six levels, never recursively scanning a user's disk.
func Identify(dir string) (string, string, string) {
	original := dir
	for i := 0; i < 6 && dir != ""; i++ {
		if b, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
			var p struct {
				Name            string            `json:"name"`
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(b, &p) == nil {
				if p.Name == "" {
					p.Name = filepath.Base(dir)
				}
				framework := "Node.js"
				for _, k := range []string{"next", "nuxt", "astro", "vite", "react", "vue", "svelte", "express"} {
					if _, ok := p.Dependencies[k]; ok {
						framework = k
						break
					}
					if _, ok := p.DevDependencies[k]; ok {
						framework = k
						break
					}
				}
				return p.Name, dir, framework
			}
		}
		for _, m := range []struct{ file, kind string }{{"go.mod", "Go"}, {"pyproject.toml", "Python"}, {"requirements.txt", "Python"}, {"Cargo.toml", "Rust"}, {"pom.xml", "Java"}, {"Gemfile", "Ruby"}, {"composer.json", "PHP"}, {".git", "Project"}} {
			if _, err := os.Stat(filepath.Join(dir, m.file)); err == nil {
				return filepath.Base(dir), dir, m.kind
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Base(original), original, ""
}

func Owned(p *process.Process) bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	name, err := p.Username()
	return err == nil && strings.EqualFold(name, u.Username)
}

func Scan(ctx context.Context, excludedPort int, filter Filter) ([]model.Service, error) {
	connections, err := gnet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return nil, err
	}
	result := []model.Service{}
	seen := map[string]bool{}
	for _, c := range connections {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if c.Status != "LISTEN" || c.Pid <= 0 || c.Pid == int32(os.Getpid()) || int(c.Laddr.Port) == excludedPort {
			continue
		}
		key := fmt.Sprintf("%d:%d", c.Pid, c.Laddr.Port)
		if seen[key] {
			continue
		}
		seen[key] = true
		p, err := process.NewProcess(c.Pid)
		if err != nil || !Owned(p) {
			continue
		}
		cwd, _ := p.CwdWithContext(ctx)
		procName, _ := p.NameWithContext(ctx)
		port := int(c.Laddr.Port)
		decision := Decide(cwd, procName, port, filter)
		if !decision.Keep {
			continue
		}
		name, dir, framework := Identify(cwd)
		if framework == "" {
			continue
		} // Unknown system listeners are deliberately not treated as projects.
		if decision.Reason == ReasonProjectMarker {
			decision.Detail = framework
		}
		birth, err := p.CreateTimeWithContext(ctx)
		if err != nil {
			continue
		}
		host := "127.0.0.1"
		if c.Laddr.IP == "::1" || c.Laddr.IP == "::" {
			host = "::1"
		}
		result = append(result, model.Service{
			Name:            name,
			Project:         name,
			Directory:       dir,
			Framework:       framework,
			Port:            port,
			URL:             "http://" + net.JoinHostPort(host, fmt.Sprint(port)),
			PID:             c.Pid,
			Birth:           birth,
			Status:          "external",
			Source:          "discovered",
			ProcessName:     procName,
			Ports:           []int{port},
			DiscoveryReason: decision.Reason,
			DiscoveryDetail: decision.Detail,
		})
	}
	return MergeByPID(result), nil
}
