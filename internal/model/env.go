package model

import (
	"regexp"
	"strings"
)

const (
	MaxEnvVars       = 40
	MaxEnvNameLength = 128
	MaxEnvValueSize  = 4096
)

var secretName = regexp.MustCompile(`(?i)(secret|token|password|passwd|credential|private|api[_-]?key)`)

func SecretName(name string) bool {
	return secretName.MatchString(strings.TrimSpace(name))
}

func NormalizeEnv(vars []EnvVar) []EnvVar {
	out := make([]EnvVar, 0, len(vars))
	seen := map[string]bool{}
	for _, item := range vars {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || len(item.Name) > MaxEnvNameLength || len(item.Value) > MaxEnvValueSize {
			continue
		}
		if seen[item.Name] {
			continue
		}
		seen[item.Name] = true
		if SecretName(item.Name) {
			item.Secret = true
		}
		out = append(out, item)
		if len(out) >= MaxEnvVars {
			break
		}
	}
	return out
}

func (v Service) Redacted() Service {
	if len(v.Env) == 0 {
		return v
	}
	env := make([]EnvVar, len(v.Env))
	copy(env, v.Env)
	for i, item := range env {
		if item.Secret || SecretName(item.Name) {
			env[i].Secret = true
			env[i].Value = ""
		}
	}
	v.Env = env
	return v
}
