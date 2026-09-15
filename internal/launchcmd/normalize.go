package launchcmd

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Normalize separates an absolute leading working-directory command from the
// command that should run there. It also accepts a missing separator after a
// quoted, existing directory so pasted terminal recipes remain useful.
func Normalize(directory, command string) (string, string, bool) {
	original := strings.TrimSpace(command)
	rest, ok := locationArguments(original)
	if !ok {
		return directory, original, false
	}
	path, remainder, ok := firstArgument(rest)
	if !ok || !filepath.IsAbs(path) {
		return directory, original, false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return directory, original, false
	}
	remainder = strings.TrimSpace(remainder)
	switch {
	case strings.HasPrefix(remainder, "&&"):
		remainder = strings.TrimSpace(remainder[2:])
	case strings.HasPrefix(remainder, ";"):
		remainder = strings.TrimSpace(remainder[1:])
	case remainder == "":
		return directory, original, false
	}
	if remainder == "" || strings.HasPrefix(remainder, "||") {
		return directory, original, false
	}
	return filepath.Clean(path), remainder, true
}

func locationArguments(command string) (string, bool) {
	trimmed := strings.TrimSpace(command)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"set-location", "pushd", "cd"} {
		if !strings.HasPrefix(lower, prefix) || (len(trimmed) > len(prefix) && !unicode.IsSpace(rune(trimmed[len(prefix)]))) {
			continue
		}
		rest := strings.TrimSpace(trimmed[len(prefix):])
		if prefix == "cd" && hasToken(rest, "/d") {
			rest = strings.TrimSpace(rest[2:])
		}
		if prefix == "set-location" {
			for _, flag := range []string{"-literalpath", "-path"} {
				if hasToken(rest, flag) {
					rest = strings.TrimSpace(rest[len(flag):])
					break
				}
			}
		}
		return rest, true
	}
	return "", false
}

func hasToken(value, token string) bool {
	return len(value) >= len(token) && strings.EqualFold(value[:len(token)], token) && (len(value) == len(token) || unicode.IsSpace(rune(value[len(token)])))
}

func firstArgument(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		for i := 1; i < len(value); i++ {
			if value[i] != quote {
				continue
			}
			if quote == '\'' && i+1 < len(value) && value[i+1] == quote {
				i++
				continue
			}
			path := value[1:i]
			if quote == '\'' {
				path = strings.ReplaceAll(path, "''", "'")
			}
			return path, value[i+1:], true
		}
		return "", "", false
	}
	for i, char := range value {
		if unicode.IsSpace(char) || char == ';' || char == '&' {
			return value[:i], value[i:], i > 0
		}
	}
	return value, "", true
}
