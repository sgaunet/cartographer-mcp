// Package crawler implements GitLab group/project discovery and metadata fetching.
package crawler

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sgaunet/cartographer-mcp/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	perPage       = 100
	httpTimeout   = 30 * time.Second
	maxReadmeLines = 500
)

// Crawler discovers projects from GitLab groups and fetches their metadata.
type Crawler struct {
	client *gitlab.Client
}

// New creates a new Crawler with rate-limit handling.
func New(token, baseURL string) (*Crawler, error) {
	httpClient := &http.Client{
		Transport: newRateLimitTransport(http.DefaultTransport),
		Timeout:   httpTimeout,
	}

	opts := []gitlab.ClientOptionFunc{
		gitlab.WithHTTPClient(httpClient),
	}
	if baseURL != "" {
		opts = append(opts, gitlab.WithBaseURL(baseURL))
	}

	client, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gitlab client: %w", err)
	}

	return &Crawler{client: client}, nil
}

// CrawlGroup discovers all projects in a group and its subgroups.
func (c *Crawler) CrawlGroup(
	ctx context.Context,
	groupPath string,
) (*models.CrawlResult, error) {
	slog.Info("crawling group", "group", groupPath)

	projects, err := c.listAllProjects(ctx, groupPath)
	if err != nil {
		return nil, fmt.Errorf("list projects for %s: %w", groupPath, err)
	}

	slog.Info("discovered projects", "group", groupPath, "count", len(projects))

	result := &models.CrawlResult{}
	for _, p := range projects {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		proj := c.fetchProjectMetadata(ctx, p)
		result.Projects = append(result.Projects, proj)
	}

	return result, nil
}

// listAllProjects lists all projects in a group including subgroups with pagination.
func (c *Crawler) listAllProjects(
	ctx context.Context,
	groupPath string,
) ([]*gitlab.Project, error) {
	var allProjects []*gitlab.Project

	opts := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
		},
		IncludeSubGroups: gitlab.Ptr(true),
	}

	for {
		projects, resp, err := c.client.Groups.ListGroupProjects(
			groupPath, opts, gitlab.WithContext(ctx),
		)
		if err != nil {
			return nil, fmt.Errorf("list group projects: %w", err)
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allProjects, nil
}

// fetchProjectMetadata fetches README, latest tag, and .cartographer.yaml.
func (c *Crawler) fetchProjectMetadata(
	ctx context.Context,
	p *gitlab.Project,
) models.Project {
	proj := convertBasicProject(p)

	// Fetch README.
	readme, err := c.fetchFileContent(ctx, p.ID, "README.md", p.DefaultBranch)
	if err != nil {
		slog.Debug("no README found",
			"project", p.PathWithNamespace, "error", err)
	} else {
		proj.ReadmeContent = truncateLines(readme, maxReadmeLines)
	}

	// Fetch latest tag.
	tags, _, err := c.client.Tags.ListTags(p.ID, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 1},
		OrderBy:     gitlab.Ptr("updated"),
		Sort:        gitlab.Ptr("desc"),
	}, gitlab.WithContext(ctx))
	if err != nil {
		slog.Debug("no tags found",
			"project", p.PathWithNamespace, "error", err)
	} else if len(tags) > 0 {
		proj.LatestVersion = tags[0].Name
	}

	// Fetch .cartographer.yaml.
	c.fetchCartographerYAML(ctx, p, &proj)

	return proj
}

func (c *Crawler) fetchCartographerYAML(
	ctx context.Context,
	p *gitlab.Project,
	proj *models.Project,
) {
	cartoFile, resp, err := c.client.RepositoryFiles.GetFile(
		p.ID,
		".cartographer.yaml",
		&gitlab.GetFileOptions{Ref: gitlab.Ptr(p.DefaultBranch)},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		logCartoError(p.PathWithNamespace, resp, err)
		return
	}

	proj.CartographerSHA = cartoFile.LastCommitID
	content, decErr := base64.StdEncoding.DecodeString(cartoFile.Content)
	if decErr != nil {
		slog.Warn(".cartographer.yaml base64 decode failed",
			"project", p.PathWithNamespace, "error", decErr)
		return
	}
	proj.RawCartographerYAML = string(content)
}

func logCartoError(path string, resp *gitlab.Response, err error) {
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		slog.Debug("no .cartographer.yaml", "project", path)
	} else {
		slog.Debug("failed to fetch .cartographer.yaml",
			"project", path, "error", err)
	}
}

// fetchFileContent retrieves a file's content decoded from base64.
func (c *Crawler) fetchFileContent(
	ctx context.Context,
	projectID int64,
	filePath, ref string,
) (string, error) {
	file, _, err := c.client.RepositoryFiles.GetFile(
		projectID,
		filePath,
		&gitlab.GetFileOptions{Ref: gitlab.Ptr(ref)},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("get file %s: %w", filePath, err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("decode base64 for %s: %w", filePath, err)
	}

	return string(content), nil
}

func convertBasicProject(p *gitlab.Project) models.Project {
	proj := models.Project{
		ID:                int(p.ID),
		Name:              p.Name,
		PathWithNamespace: p.PathWithNamespace,
		Description:       p.Description,
		DefaultBranch:     p.DefaultBranch,
		WebURL:            p.WebURL,
		Archived:          p.Archived,
		FetchedAt:         time.Now().UTC(),
	}
	if p.LastActivityAt != nil {
		proj.LastActivityAt = *p.LastActivityAt
	}
	return proj
}

func truncateLines(content string, maxLines int) string {
	lines := strings.SplitN(content, "\n", maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}
