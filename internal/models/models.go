// Package models defines shared data structures used across packages.
package models

import "time"

// Catalog is the root object persisted to catalog.json.
type Catalog struct {
	SchemaVersion     int          `json:"schema_version"`
	RefreshedAt       time.Time    `json:"refreshed_at"`
	RefreshDurationMs int64        `json:"refresh_duration_ms"`
	GitLabURI         string       `json:"gitlab_uri"`
	Groups            []string     `json:"groups"`
	Projects          []Project    `json:"projects"`
	Diagnostics       []Diagnostic `json:"diagnostics,omitempty"`
}

// Project is a discovered GitLab project with merged metadata.
type Project struct {
	ID                  int              `json:"id"`
	Name                string           `json:"name"`
	PathWithNamespace   string           `json:"path_with_namespace"`
	Description         string           `json:"description,omitempty"`
	DefaultBranch       string           `json:"default_branch"`
	WebURL              string           `json:"web_url"`
	Archived            bool             `json:"archived"`
	LastActivityAt      time.Time        `json:"last_activity_at"`
	LatestVersion       string           `json:"latest_version,omitempty"`
	ReadmeContent       string           `json:"readme_content,omitempty"`
	FetchedAt           time.Time        `json:"fetched_at"`
	CartographerSHA     string           `json:"cartographer_sha,omitempty"`
	RawCartographerYAML string           `json:"-"`
	Service             *ServiceMetadata `json:"service,omitempty"`
}

// ServiceMetadata holds human-defined metadata from .cartographer.yaml.
type ServiceMetadata struct {
	SchemaVersion      int          `json:"schema_version"                yaml:"schema_version"`
	Name               string       `json:"name"                          yaml:"name"`
	Type               string       `json:"type"                          yaml:"type"`
	Lifecycle          string       `json:"lifecycle"                     yaml:"lifecycle"`
	Owner              string       `json:"owner,omitempty"               yaml:"owner"`
	Description        string       `json:"description,omitempty"         yaml:"description"`
	Tags               []string     `json:"tags,omitempty"                yaml:"tags"`
	Dependencies       []Dependency `json:"dependencies,omitempty"        yaml:"dependencies"`
	Outputs            []Output     `json:"outputs,omitempty"             yaml:"outputs"`
	ValidationWarnings []string     `json:"validation_warnings,omitempty" yaml:"-"`
}

// Dependency is a declared relationship from one service to another.
type Dependency struct {
	Service string `json:"service"        yaml:"service"`
	Type    string `json:"type,omitempty" yaml:"type"`
}

// Output is something a service produces or exposes.
type Output struct {
	Name        string `json:"name"                  yaml:"name"`
	Type        string `json:"type"                  yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description"`
}

// CrawlResult holds the projects discovered from a single group crawl.
type CrawlResult struct {
	Projects []Project
}

// Diagnostic is a warning or error from a refresh operation.
type Diagnostic struct {
	ProjectPath string    `json:"project_path"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}
