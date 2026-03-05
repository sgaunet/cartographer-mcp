// Package config handles configuration loading from environment variables and YAML files.
package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrMissingToken is returned when the GITLAB_TOKEN env var is not set.
	ErrMissingToken = errors.New("GITLAB_TOKEN environment variable is required")
	// ErrNoHomeDir is returned when the home directory cannot be determined.
	ErrNoHomeDir = errors.New("cannot determine home directory")
)

const defaultGitLabURI = "https://gitlab.com/"

// Config holds the runtime configuration for the server.
type Config struct {
	GitLabURI   string
	GitLabToken string
	Groups      []string
	CacheDir    string
}

// configFile represents the YAML configuration file structure.
type configFile struct {
	Groups    []string `yaml:"groups"`
	GitLabURI string   `yaml:"gitlab_uri"`
	CacheDir  string   `yaml:"cache_dir"`
}

// Load reads configuration from a YAML config file and environment variables.
// Environment variables take precedence over config file values.
func Load() (*Config, error) {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return nil, ErrMissingToken
	}

	file := loadConfigFile()

	uri := resolveURI(file)
	groups := resolveGroups(file)
	cacheDir, err := resolveCacheDir(file)
	if err != nil {
		return nil, err
	}

	return &Config{
		GitLabURI:   uri,
		GitLabToken: token,
		Groups:      groups,
		CacheDir:    cacheDir,
	}, nil
}

func resolveURI(file *configFile) string {
	if uri := os.Getenv("GITLAB_URI"); uri != "" {
		return uri
	}
	if file != nil && file.GitLabURI != "" {
		return file.GitLabURI
	}
	return defaultGitLabURI
}

func resolveGroups(file *configFile) []string {
	if g := os.Getenv("CARTOGRAPHER_GROUPS"); g != "" {
		var groups []string
		for s := range strings.SplitSeq(g, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				groups = append(groups, s)
			}
		}
		return groups
	}
	if file != nil {
		return file.Groups
	}
	return nil
}

func resolveCacheDir(file *configFile) (string, error) {
	if dir := os.Getenv("CARTOGRAPHER_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	if file != nil && file.CacheDir != "" {
		return os.ExpandEnv(file.CacheDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ErrNoHomeDir
	}
	return filepath.Join(home, ".config", "cartographer", "cache"), nil
}

// loadConfigFile attempts to find and parse the config file.
func loadConfigFile() *configFile {
	configPath := os.Getenv("CARTOGRAPHER_CONFIG")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		configPath = filepath.Join(home, ".config", "cartographer", "config.yaml")
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // config path is user-controlled by design
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("failed to read config file",
				"path", configPath, "error", err.Error())
		}
		return nil
	}

	var cf configFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		slog.Warn("failed to parse config file",
			"path", configPath, "error", err.Error())
		return nil
	}

	slog.Info("loaded config file",
		"path", configPath, "groups", len(cf.Groups))
	return &cf
}
