package datasource

import (
	"fmt"
	"gcliv2/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveID(t *testing.T) {
	// Mock Grafana API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/datasources" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[{"id":1, "name":"Prometheus"}, {"id":2, "name":"InfluxDB"}]`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// Setup temp config
	tmpDir, _ := os.MkdirTemp("", "gcli-ds-test-*")
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

	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"Prometheus", 1, false},
		{"1", 1, false},
		{"2", 2, false},
		{"Unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := ResolveID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.expected {
				t.Errorf("ResolveID() = %v, want %v", id, tt.expected)
			}
		})
	}
}
