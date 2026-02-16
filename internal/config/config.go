package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type fileConfig struct {
	Active    string             `yaml:"active"`
	ActiveOrg string             `yaml:"active_org"`
	Profiles  map[string]Profile `yaml:"profiles"`
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".gcli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func load() (*fileConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return &fileConfig{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

func save(cfg *fileConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, out, 0o600)
}

// SaveProfile adds or updates a profile.
func SaveProfile(p Profile) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	cfg.Profiles[p.Name] = p
	return save(cfg)
}

func LoadAll() (map[string]Profile, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}
	return cfg.Profiles, nil
}

func SetActive(name string) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %s does not exist", name)
	}
	cfg.Active = name
	return save(cfg)
}

func GetActive() (*Profile, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}
	if cfg.Active == "" {
		return nil, nil
	}
	p, ok := cfg.Profiles[cfg.Active]
	if !ok {
		return nil, fmt.Errorf("active profile %s not found", cfg.Active)
	}
	return &p, nil
}

// SetActiveOrg stores the selected organization ID.
func SetActiveOrg(orgID string) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	cfg.ActiveOrg = orgID
	return save(cfg)
}

// GetActiveOrg returns the currently selected organization ID.
func GetActiveOrg() (string, error) {
	cfg, err := load()
	if err != nil {
		return "", err
	}
	return cfg.ActiveOrg, nil
}
