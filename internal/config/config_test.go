package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManagement(t *testing.T) {
	// Setup temporary config path
	tmpDir, err := os.MkdirTemp("", "gcli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpCfg := filepath.Join(tmpDir, "config.yaml")
	os.Setenv("GCLI_CONFIG_PATH", tmpCfg)
	defer os.Unsetenv("GCLI_CONFIG_PATH")

	// Test SaveProfile
	p := Profile{
		Name: "test-profile",
		URL:  "http://localhost:3000",
		User: "admin",
		Pass: "admin",
	}

	err = SaveProfile(p)
	if err != nil {
		t.Errorf("SaveProfile failed: %v", err)
	}

	// Test LoadAll
	profiles, err := LoadAll()
	if err != nil {
		t.Errorf("LoadAll failed: %v", err)
	}
	if _, ok := profiles["test-profile"]; !ok {
		t.Errorf("profile not found in LoadAll")
	}

	// Test SetActive
	err = SetActive("test-profile")
	if err != nil {
		t.Errorf("SetActive failed: %v", err)
	}

	// Test GetActive
	active, err := GetActive()
	if err != nil {
		t.Errorf("GetActive failed: %v", err)
	}
	if active == nil || active.Name != "test-profile" {
		t.Errorf("expected active profile 'test-profile', got %v", active)
	}

	// Test SetActiveOrg
	err = SetActiveOrg("1")
	if err != nil {
		t.Errorf("SetActiveOrg failed: %v", err)
	}

	// Test GetActiveOrg
	org, err := GetActiveOrg()
	if err != nil {
		t.Errorf("GetActiveOrg failed: %v", err)
	}
	if org != "1" {
		t.Errorf("expected org '1', got %s", org)
	}
}
