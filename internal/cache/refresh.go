package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sgaunet/cartographer-mcp/internal/models"
	"github.com/sgaunet/cartographer-mcp/internal/schema"
)

// Refresher handles crawling groups and updating the cache.
type Refresher struct {
	store   *Store
	crawler GroupCrawler
}

// GroupCrawler is the interface the crawler must implement.
type GroupCrawler interface {
	CrawlGroup(ctx context.Context, groupPath string) (*models.CrawlResult, error)
}

// NewRefresher creates a new Refresher.
func NewRefresher(store *Store, crawler GroupCrawler) *Refresher {
	return &Refresher{store: store, crawler: crawler}
}

// RefreshResult contains the outcome of a refresh operation.
type RefreshResult struct {
	RefreshedAt              time.Time
	DurationMs               int64
	ProjectsDiscovered       int
	ProjectsWithCartographer int
	Diagnostics              []models.Diagnostic
}

// Refresh crawls the given groups and updates the cache.
func (r *Refresher) Refresh(
	ctx context.Context,
	gitlabURI string,
	groups []string,
) (*RefreshResult, error) {
	start := time.Now()

	allProjects, diagnostics := r.crawlAllGroups(ctx, groups)
	cartoCount := r.parseCartographerFiles(allProjects, &diagnostics)
	finalProjects := r.mergeWithPrevious(allProjects, &diagnostics)

	catalog := &models.Catalog{
		SchemaVersion:     1,
		RefreshedAt:       start.UTC(),
		RefreshDurationMs: time.Since(start).Milliseconds(),
		GitLabURI:         gitlabURI,
		Groups:            groups,
		Projects:          finalProjects,
		Diagnostics:       diagnostics,
	}

	r.store.SetCatalog(catalog)
	if err := r.store.Save(); err != nil {
		return nil, fmt.Errorf("save catalog: %w", err)
	}

	return &RefreshResult{
		RefreshedAt:              start.UTC(),
		DurationMs:               time.Since(start).Milliseconds(),
		ProjectsDiscovered:       len(finalProjects),
		ProjectsWithCartographer: cartoCount,
		Diagnostics:              diagnostics,
	}, nil
}

func (r *Refresher) crawlAllGroups(
	ctx context.Context,
	groups []string,
) ([]models.Project, []models.Diagnostic) {
	var allProjects []models.Project
	var diagnostics []models.Diagnostic

	for _, group := range groups {
		result, err := r.crawler.CrawlGroup(ctx, group)
		if err != nil {
			slog.Error("failed to crawl group", "group", group, "error", err)
			diagnostics = append(diagnostics, models.Diagnostic{
				ProjectPath: group,
				Level:       "error",
				Message:     "failed to crawl group: " + err.Error(),
				Timestamp:   time.Now().UTC(),
			})
			continue
		}
		allProjects = append(allProjects, result.Projects...)
	}

	return allProjects, diagnostics
}

func (r *Refresher) parseCartographerFiles(
	projects []models.Project,
	diagnostics *[]models.Diagnostic,
) int {
	cartoCount := 0
	for i := range projects {
		p := &projects[i]
		if p.RawCartographerYAML == "" {
			continue
		}

		sm, err := schema.ParseCartographerYAML([]byte(p.RawCartographerYAML))
		if err != nil {
			*diagnostics = append(*diagnostics, models.Diagnostic{
				ProjectPath: p.PathWithNamespace,
				Level:       "warning",
				Message:     ".cartographer.yaml: " + err.Error(),
				Timestamp:   time.Now().UTC(),
			})
			continue
		}

		for _, w := range sm.ValidationWarnings {
			*diagnostics = append(*diagnostics, models.Diagnostic{
				ProjectPath: p.PathWithNamespace,
				Level:       "warning",
				Message:     ".cartographer.yaml: " + w,
				Timestamp:   time.Now().UTC(),
			})
		}

		p.Service = sm
		cartoCount++
	}
	return cartoCount
}

func (r *Refresher) mergeWithPrevious(
	newProjects []models.Project,
	diagnostics *[]models.Diagnostic,
) []models.Project {
	previousIdx := make(map[string]models.Project)
	for _, p := range r.store.GetCatalog().Projects {
		previousIdx[p.PathWithNamespace] = p
	}

	seen := make(map[string]bool)
	for _, p := range newProjects {
		seen[p.PathWithNamespace] = true
	}

	for path, prev := range previousIdx {
		if !seen[path] {
			slog.Info("retaining stale data for missing project", "project", path)
			*diagnostics = append(*diagnostics, models.Diagnostic{
				ProjectPath: path,
				Level:       "warning",
				Message:     "project not found in refresh, retained stale data",
				Timestamp:   time.Now().UTC(),
			})
			newProjects = append(newProjects, prev)
		}
	}

	return newProjects
}
