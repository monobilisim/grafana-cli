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

func TestDataSourceCommands(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/datasources" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[{"id":1, "name":"PromTest", "type":"prometheus", "url":"http://localhost:9090"}]`)
		}
	}))
	defer ts.Close()

	tmpDir, _ := os.MkdirTemp("", "gcli-ds-cmd-test-*")
	defer os.RemoveAll(tmpDir)
	tmpCfg := filepath.Join(tmpDir, "config.yaml")
	os.Setenv("GCLI_CONFIG_PATH", tmpCfg)
	defer os.Unsetenv("GCLI_CONFIG_PATH")

	config.SaveProfile(config.Profile{Name: "test", URL: ts.URL, User: "a", Pass: "a"})
	config.SetActive("test")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"ds", "list"})
	rootCmd.Execute()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("PromTest")) {
		t.Errorf("expected 'PromTest' in output, got %s", buf.String())
	}
}
