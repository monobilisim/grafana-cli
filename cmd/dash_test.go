package cmd

import (
	"bytes"
	"fmt"
	"gcliv2/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDashboardCommands(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[{"uid":"abc", "title":"Test Dash", "type":"dash-db"}]`)
		}
	}))
	defer ts.Close()

	tmpDir, _ := os.MkdirTemp("", "gcli-dash-cmd-test-*")
	defer os.RemoveAll(tmpDir)
	tmpCfg := filepath.Join(tmpDir, "config.yaml")
	os.Setenv("GCLI_CONFIG_PATH", tmpCfg)
	defer os.Unsetenv("GCLI_CONFIG_PATH")

	config.SaveProfile(config.Profile{Name: "test", URL: ts.URL, User: "a", Pass: "a"})
	config.SetActive("test")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"dash", "list"})
	rootCmd.Execute()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("Test Dash")) {
		t.Errorf("expected 'Test Dash' in output, got %s", buf.String())
	}
}
