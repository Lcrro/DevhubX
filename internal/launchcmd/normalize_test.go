package launchcmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalize(t *testing.T) {
	dir := t.TempDir()
	quoted := `"` + dir + `"`
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"semicolon", "cd " + quoted + `; npm run dev`, "npm run dev"},
		{"and", "cd " + quoted + ` && npm run dev`, "npm run dev"},
		{"missing separator", "cd " + quoted + ` powershell.exe -File .\\start.ps1`, `powershell.exe -File .\\start.ps1`},
		{"set location", "Set-Location -LiteralPath " + quoted + `; go run ./cmd/app`, "go run ./cmd/app"},
		{"pushd", "pushd " + quoted + ` && python app.py`, "python app.py"},
	}
	if runtime.GOOS == "windows" {
		tests = append(tests, struct{ name, command, want string }{"cmd cd", "cd /d " + quoted + ` && npm start`, "npm start"})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotDir, gotCommand, changed := Normalize("", test.command)
			if !changed || gotDir != filepath.Clean(dir) || gotCommand != test.want {
				t.Fatalf("got %q, %q, %v", gotDir, gotCommand, changed)
			}
		})
	}
}

func TestNormalizeRejectsUnsafeOrAmbiguousInput(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	for _, command := range []string{
		"npm run dev",
		"cd " + missing + " && npm run dev",
		"cd " + dir,
		"cd " + dir + " || npm run fallback",
		"cd relative && npm run dev",
	} {
		if _, got, changed := Normalize(dir, command); changed || got != command {
			t.Fatalf("unexpected normalization of %q", command)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "child"), 0700); err != nil {
		t.Fatal(err)
	}
}
