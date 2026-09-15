package discovery

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"devhub/internal/model"
)

const (
	ReasonProjectMarker      = "project-marker"
	ReasonIncludeDirectory   = "include-directory"
	ReasonIncludeProcess     = "include-process"
	ReasonIncludePort        = "include-port"
	ReasonMergedPorts        = "merged-ports"
	ReasonSystemInstall      = "system-install"
	ReasonExcludeDirectory   = "exclude-directory"
	ReasonExcludeProcess     = "exclude-process"
	ReasonExcludePort        = "exclude-port"
	ReasonIncludeDirMiss     = "include-directory-miss"
	ReasonIncludeProcessMiss = "include-process-miss"
	ReasonIncludePortMiss    = "include-port-miss"
)

var preferredWebPorts = []int{80, 443, 3000, 3001, 4173, 5173, 5174, 8000, 8080, 8081, 8888, 9000, 9090}

// Filter is the user-controlled discovery policy. Empty include lists mean no
// extra allowlist; exclude lists always apply after the system-install skip.
type Filter struct {
	IncludeDirectories []string
	ExcludeDirectories []string
	IncludeProcesses   []string
	ExcludeProcesses   []string
	IncludePorts       []int
	ExcludePorts       []int
}

func FilterFromSettings(settings model.Settings) Filter {
	settings = settings.Normalized()
	return Filter{
		IncludeDirectories: settings.IncludeDirectories,
		ExcludeDirectories: settings.ExcludeDirectories,
		IncludeProcesses:   settings.IncludeProcesses,
		ExcludeProcesses:   settings.ExcludeProcesses,
		IncludePorts:       settings.IncludePorts,
		ExcludePorts:       settings.ExcludePorts,
	}
}

func (f Filter) Equal(other Filter) bool {
	return stringListsEqual(f.IncludeDirectories, other.IncludeDirectories) &&
		stringListsEqual(f.ExcludeDirectories, other.ExcludeDirectories) &&
		stringListsEqual(f.IncludeProcesses, other.IncludeProcesses) &&
		stringListsEqual(f.ExcludeProcesses, other.ExcludeProcesses) &&
		intListsEqual(f.IncludePorts, other.IncludePorts) &&
		intListsEqual(f.ExcludePorts, other.ExcludePorts)
}

func stringListsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intListsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type Decision struct {
	Keep   bool
	Reason string
	Detail string
}

func Decide(cwd, process string, port int, filter Filter) Decision {
	if ManagedInstallPath(cwd) {
		return Decision{Reason: ReasonSystemInstall, Detail: cwd}
	}
	if matchProcess(process, filter.ExcludeProcesses) {
		return Decision{Reason: ReasonExcludeProcess, Detail: process}
	}
	if containsPort(filter.ExcludePorts, port) {
		return Decision{Reason: ReasonExcludePort, Detail: strconv.Itoa(port)}
	}
	if root := matchingRoot(cwd, filter.ExcludeDirectories); root != "" {
		return Decision{Reason: ReasonExcludeDirectory, Detail: root}
	}
	if len(filter.IncludePorts) > 0 && !containsPort(filter.IncludePorts, port) {
		return Decision{Reason: ReasonIncludePortMiss, Detail: strconv.Itoa(port)}
	}
	if len(filter.IncludeProcesses) > 0 && !matchProcess(process, filter.IncludeProcesses) {
		return Decision{Reason: ReasonIncludeProcessMiss, Detail: process}
	}
	if len(filter.IncludeDirectories) > 0 {
		root := matchingRoot(cwd, filter.IncludeDirectories)
		if root == "" {
			return Decision{Reason: ReasonIncludeDirMiss, Detail: cwd}
		}
		return Decision{Keep: true, Reason: ReasonIncludeDirectory, Detail: root}
	}
	if len(filter.IncludeProcesses) > 0 {
		return Decision{Keep: true, Reason: ReasonIncludeProcess, Detail: process}
	}
	if len(filter.IncludePorts) > 0 {
		return Decision{Keep: true, Reason: ReasonIncludePort, Detail: strconv.Itoa(port)}
	}
	return Decision{Keep: true, Reason: ReasonProjectMarker}
}

func ServiceExcluded(v model.Service, filter Filter) bool {
	if v.Source != "discovered" {
		return false
	}
	if ManagedInstallPath(v.Directory) {
		return true
	}
	ports := v.Ports
	if len(ports) == 0 && v.Port > 0 {
		ports = []int{v.Port}
	}
	if len(ports) == 0 {
		return !Decide(v.Directory, v.ProcessName, v.Port, filter).Keep
	}
	for _, port := range ports {
		if Decide(v.Directory, v.ProcessName, port, filter).Keep {
			return false
		}
	}
	return true
}

func MergeByPID(items []model.Service) []model.Service {
	groups := map[int32][]model.Service{}
	order := []int32{}
	orphans := []model.Service{}
	for _, item := range items {
		if item.PID <= 0 {
			item.Ports = uniquePorts(append(item.Ports, item.Port))
			orphans = append(orphans, item)
			continue
		}
		if _, ok := groups[item.PID]; !ok {
			order = append(order, item.PID)
		}
		groups[item.PID] = append(groups[item.PID], item)
	}
	result := make([]model.Service, 0, len(items))
	for _, pid := range order {
		result = append(result, mergeGroup(groups[pid]))
	}
	result = append(result, orphans...)
	sort.Slice(result, func(i, j int) bool { return result[i].Port < result[j].Port })
	return result
}

func mergeGroup(group []model.Service) model.Service {
	ports := make([]int, 0, len(group))
	byPort := map[int]model.Service{}
	for _, item := range group {
		ports = append(ports, item.Port)
		ports = append(ports, item.Ports...)
		byPort[item.Port] = item
	}
	ports = uniquePorts(ports)
	primary := ChoosePrimaryPort(ports, 0)
	base, ok := byPort[primary]
	if !ok {
		base = group[0]
	}
	base.Ports = ports
	if base.Port != primary {
		base.URL = URLForPort(base.URL, primary)
		base.Port = primary
	}
	if len(ports) > 1 && base.DiscoveryReason == ReasonProjectMarker {
		base.DiscoveryReason = ReasonMergedPorts
		base.DiscoveryDetail = joinPorts(ports)
	}
	return base
}

func ChoosePrimaryPort(ports []int, preferred int) int {
	ports = uniquePorts(ports)
	if len(ports) == 0 {
		return preferred
	}
	if containsPort(ports, preferred) {
		return preferred
	}
	for _, port := range preferredWebPorts {
		if containsPort(ports, port) {
			return port
		}
	}
	return ports[0]
}

func URLForPort(raw string, port int) string {
	if port < 1 {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	host := u.Hostname()
	if host == "" {
		host, _, _ = net.SplitHostPort(u.Host)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u.String()
}

func matchingRoot(dir string, roots []string) string {
	for _, root := range roots {
		if pathWithinInsensitive(root, dir) {
			return root
		}
	}
	return ""
}

func pathWithinInsensitive(root, candidate string) bool {
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
		candidate = strings.ToLower(candidate)
	}
	return pathWithin(root, candidate)
}

func matchProcess(name string, patterns []string) bool {
	normalized := normalizeProcess(name)
	if normalized == "" {
		return false
	}
	for _, pattern := range patterns {
		if normalizeProcess(pattern) == normalized {
			return true
		}
	}
	return false
}

func normalizeProcess(name string) string {
	name = strings.TrimSpace(strings.ToLower(filepath.Base(name)))
	return strings.TrimSuffix(name, ".exe")
}

func containsPort(ports []int, port int) bool {
	for _, candidate := range ports {
		if candidate == port {
			return true
		}
	}
	return false
}

func uniquePorts(ports []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(ports))
	for _, port := range ports {
		if port < 1 || port > 65535 || seen[port] {
			continue
		}
		seen[port] = true
		out = append(out, port)
	}
	sort.Ints(out)
	return out
}

func joinPorts(ports []int) string {
	parts := make([]string, len(ports))
	for i, port := range ports {
		parts[i] = strconv.Itoa(port)
	}
	return strings.Join(parts, ", ")
}
