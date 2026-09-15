package cover

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalURL(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1:3000", "http://localhost:8080", "https://[::1]:4000"} {
		if !LocalURL(u) {
			t.Error("rejected", u)
		}
	}
	for _, u := range []string{"https://example.com", "http://127.0.0.1.evil.test", "file:///etc/passwd", "http://user:password@localhost", "http://192.168.1.1", "javascript:alert(1)"} {
		if LocalURL(u) {
			t.Error("accepted", u)
		}
	}
}
func TestCaptureIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_BROWSER") != "1" {
		t.Skip("set DEVHUB_TEST_BROWSER=1 to run real browser capture")
	}
	if Browser() == "" {
		t.Fatal("browser not found")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><h1>DevHub screenshot test</h1></body></html>`))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "cover.png")
	if err := Capture(context.Background(), srv.URL, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(data, []byte{137, 80, 78, 71}) {
		t.Fatal("invalid PNG", err)
	}
}
