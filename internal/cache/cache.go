// Package cache provides JSON file persistence for the service catalog.
package cache

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/sgaunet/cartographer-mcp/internal/models"
)

// Store manages JSON file persistence for the service catalog.
type Store struct {
	dir     string
	catalog *models.Catalog
}

// NewStore creates a new cache store with the given directory.
func NewStore(dir string) *Store {
	return &Store{
		dir: dir,
		catalog: &models.Catalog{
			SchemaVersion: 1,
			Projects:      []models.Project{},
		},
	}
}

// GetCatalog returns the current catalog.
func (s *Store) GetCatalog() *models.Catalog {
	return s.catalog
}

// SetCatalog replaces the current catalog.
func (s *Store) SetCatalog(cat *models.Catalog) {
	s.catalog = cat
}

// CacheAgeSeconds returns seconds since the last refresh.
func (s *Store) CacheAgeSeconds() int {
	if s.catalog.RefreshedAt.IsZero() {
		return -1
	}
	return int(time.Since(s.catalog.RefreshedAt).Seconds())
}

// Load reads the catalog from disk. If the file is missing or corrupt,
// the store starts with an empty catalog.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.catalogPath())
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("no existing catalog found, starting empty")
			return nil
		}
		slog.Warn("failed to read catalog file, starting empty", "error", err)
		return nil
	}

	var cat models.Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		slog.Warn("catalog file is corrupt, starting empty", "error", err)
		return nil
	}

	s.catalog = &cat
	slog.Info("loaded catalog",
		"projects", len(cat.Projects),
		"refreshed_at", cat.RefreshedAt)
	return nil
}

// Save writes the catalog to disk atomically via temp file + rename.
func (s *Store) Save() error {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(s.catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}

	tmpFile := s.catalogPath() + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("write temp catalog: %w", err)
	}

	if err := os.Rename(tmpFile, s.catalogPath()); err != nil {
		return fmt.Errorf("rename catalog: %w", err)
	}

	return nil
}

func (s *Store) catalogPath() string {
	return filepath.Join(s.dir, "catalog.json")
}
