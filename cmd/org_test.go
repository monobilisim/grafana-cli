package cmd

import (
	"bytes"
	"fmt"
	"gcli/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOrgCommands(t *testing.T) {
	// Mock Grafana API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/api/orgs" {
				fmt.Fprintln(w, `[{"id":1, "name":"Main Org."}]`)
			}
		case http.MethodPost:
			if r.URL.Path == "/api/orgs" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"message":"Organization created", "orgId":2}`)
			}
		}
	}))
	defer ts.Close()

	// Setup temp config
	tmpDir, _ := os.MkdirTemp("", "gcli-org-test-*")
	defer os.RemoveAll(tmpDir)
	tmpCfg := filepath.Join(tmpDir, "config.yaml")
	os.Setenv("GCLI_CONFIG_PATH", tmpCfg)
	defer os.Unsetenv("GCLI_CONFIG_PATH")

	config.SaveProfile(config.Profile{
		Name: "test",
		URL:  ts.URL,
		User: "admin",
		Pass: "admin",
	})
	config.SetActive("test")

	// Test 'org list'
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"org", "list"})
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("org list failed: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Main Org.")) {
		t.Errorf("expected 'Main Org.' in output, got %s", buf.String())
	}
}
