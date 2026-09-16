package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Settings controls the local DevHub workspace. Values are persisted in SQLite
// and deliberately kept small so the file remains portable and easy to back up.
type Settings struct {
	Language            string        `json:"language"`
	AutoScan            bool          `json:"autoScan"`
	ScanIntervalSeconds int           `json:"scanIntervalSeconds"`
	IncludeDirectories  []string      `json:"includeDirectories"`
	ExcludeDirectories  []string      `json:"excludeDirectories"`
	IncludeProcesses    []string      `json:"includeProcesses"`
	ExcludeProcesses    []string      `json:"excludeProcesses"`
	IncludePorts        []int         `json:"includePorts"`
	ExcludePorts        []int         `json:"excludePorts"`
	Theme               string        `json:"theme"`
	Accent              string        `json:"accent"`
	Density             string        `json:"density"`
	ServiceView         string        `json:"serviceView"`
	ProjectPrefs        []ProjectPref `json:"projectPrefs"`
}

type ProjectPref struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
}

const (
	MinScanIntervalSeconds = 5
	MaxScanIntervalSeconds = 300
	MaxDiscoveryRules      = 50
	MaxDiscoveryPathLength = 512
	MaxProcessNameLength   = 128
)

func DefaultSettings() Settings {
	return Settings{
		Language:            "en",
		AutoScan:            true,
		ScanIntervalSeconds: 10,
		IncludeDirectories:  []string{},
		ExcludeDirectories:  []string{},
		IncludeProcesses:    []string{},
		ExcludeProcesses:    []string{},
		IncludePorts:        []int{},
		ExcludePorts:        []int{},
		Theme:               "dark",
		Accent:              "lime",
		Density:             "comfortable",
		ServiceView:         "cards",
		ProjectPrefs:        []ProjectPref{},
	}
}

func (s Settings) Normalized() Settings {
	defaults := DefaultSettings()
	if s.Language != "zh" && s.Language != "en" {
		s.Language = defaults.Language
	}
	if s.ScanIntervalSeconds < MinScanIntervalSeconds {
		s.ScanIntervalSeconds = MinScanIntervalSeconds
	}
	if s.ScanIntervalSeconds > MaxScanIntervalSeconds {
		s.ScanIntervalSeconds = MaxScanIntervalSeconds
	}
	s.IncludeDirectories = cleanPaths(s.IncludeDirectories)
	s.ExcludeDirectories = cleanPaths(s.ExcludeDirectories)
	s.IncludeProcesses = cleanStrings(s.IncludeProcesses, MaxProcessNameLength)
	s.ExcludeProcesses = cleanStrings(s.ExcludeProcesses, MaxProcessNameLength)
	s.IncludePorts = cleanPorts(s.IncludePorts)
	s.ExcludePorts = cleanPorts(s.ExcludePorts)
	if s.Theme != "dark" && s.Theme != "light" {
		s.Theme = defaults.Theme
	}
	if s.Accent != "lime" && s.Accent != "cyan" && s.Accent != "violet" && s.Accent != "amber" {
		s.Accent = defaults.Accent
	}
	if s.Density != "comfortable" && s.Density != "compact" {
		s.Density = defaults.Density
	}
	if s.ServiceView != "cards" && s.ServiceView != "list" {
		s.ServiceView = defaults.ServiceView
	}
	s.ProjectPrefs = cleanProjectPrefs(s.ProjectPrefs)
	return s
}

func (s Settings) Validate() error {
	if s.Language != "zh" && s.Language != "en" {
		return fmt.Errorf("language must be zh or en")
	}
	if s.ScanIntervalSeconds < MinScanIntervalSeconds || s.ScanIntervalSeconds > MaxScanIntervalSeconds {
		return fmt.Errorf("扫描间隔必须在 %d–%d 秒之间", MinScanIntervalSeconds, MaxScanIntervalSeconds)
	}
	for _, group := range [][]string{s.IncludeDirectories, s.ExcludeDirectories} {
		if len(group) > MaxDiscoveryRules {
			return fmt.Errorf("每类发现规则最多 %d 条", MaxDiscoveryRules)
		}
		for _, dir := range group {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}
			if !filepath.IsAbs(dir) {
				return fmt.Errorf("发现规则中的目录必须是绝对路径")
			}
			if len(dir) > MaxDiscoveryPathLength {
				return fmt.Errorf("发现规则中的目录过长")
			}
		}
	}
	for _, group := range [][]string{s.IncludeProcesses, s.ExcludeProcesses} {
		if len(group) > MaxDiscoveryRules {
			return fmt.Errorf("每类发现规则最多 %d 条", MaxDiscoveryRules)
		}
		for _, name := range group {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if len(name) > MaxProcessNameLength {
				return fmt.Errorf("发现规则中的进程名过长")
			}
		}
	}
	for _, group := range [][]int{s.IncludePorts, s.ExcludePorts} {
		if len(group) > MaxDiscoveryRules {
			return fmt.Errorf("每类发现规则最多 %d 条", MaxDiscoveryRules)
		}
		for _, port := range group {
			if port < 1 || port > 65535 {
				return fmt.Errorf("发现规则中的端口必须在 1–65535 之间")
			}
		}
	}
	if s.Theme != "dark" && s.Theme != "light" && s.Theme != "" {
		return fmt.Errorf("theme must be dark or light")
	}
	if s.Accent != "lime" && s.Accent != "cyan" && s.Accent != "violet" && s.Accent != "amber" && s.Accent != "" {
		return fmt.Errorf("accent is not supported")
	}
	if s.Density != "comfortable" && s.Density != "compact" && s.Density != "" {
		return fmt.Errorf("density must be comfortable or compact")
	}
	if s.ServiceView != "cards" && s.ServiceView != "list" && s.ServiceView != "" {
		return fmt.Errorf("serviceView must be cards or list")
	}
	if len(s.ProjectPrefs) > MaxDiscoveryRules {
		return fmt.Errorf("项目外观最多 %d 条", MaxDiscoveryRules)
	}
	return nil
}

func cleanProjectPrefs(values []ProjectPref) []ProjectPref {
	out := make([]ProjectPref, 0, len(values))
	seen := map[string]bool{}
	allowedColor := map[string]bool{"lime": true, "cyan": true, "violet": true, "amber": true, "rose": true, "gray": true}
	allowedIcon := map[string]bool{"folder": true, "code": true, "server": true, "globe": true, "database": true}
	for _, item := range values {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || len(item.Name) > 100 || seen[strings.ToLower(item.Name)] {
			continue
		}
		seen[strings.ToLower(item.Name)] = true
		if !allowedColor[item.Color] {
			item.Color = "lime"
		}
		if !allowedIcon[item.Icon] {
			item.Icon = "folder"
		}
		out = append(out, item)
		if len(out) >= MaxDiscoveryRules {
			break
		}
	}
	return out
}

func cleanPaths(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !filepath.IsAbs(value) || len(value) > MaxDiscoveryPathLength {
			continue
		}
		value = filepath.Clean(value)
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if len(out) >= MaxDiscoveryRules {
			break
		}
	}
	return out
}

func cleanStrings(values []string, maxLen int) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > maxLen {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if len(out) >= MaxDiscoveryRules {
			break
		}
	}
	return out
}

func cleanPorts(values []int) []int {
	out := make([]int, 0, len(values))
	seen := map[int]bool{}
	for _, port := range values {
		if port < 1 || port > 65535 || seen[port] {
			continue
		}
		seen[port] = true
		out = append(out, port)
		if len(out) >= MaxDiscoveryRules {
			break
		}
	}
	return out
}
