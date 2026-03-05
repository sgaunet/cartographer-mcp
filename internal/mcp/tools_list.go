package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const emptyCacheMsg = `{"message":"No services in cache. ` +
	`Run refresh_cache to populate.","cache_age_seconds":-1}`

func (s *Server) registerListServices() {
	tool := mcp.NewTool("list_services",
		mcp.WithDescription(
			"List all discovered services, optionally filtered",
		),
		mcp.WithString("type",
			mcp.Description("Filter by service type"),
		),
		mcp.WithString("lifecycle",
			mcp.Description("Filter by lifecycle stage"),
		),
		mcp.WithString("tag",
			mcp.Description("Filter by tag (exact match)"),
		),
		mcp.WithBoolean("include_archived",
			mcp.Description("Include archived projects (default: false)"),
		),
	)

	s.addTool(tool, s.handleListServices)
}

type serviceEntry struct {
	Name                string   `json:"name"`
	PathWithNamespace   string   `json:"path_with_namespace"`
	Type                string   `json:"type,omitempty"`
	Lifecycle           string   `json:"lifecycle,omitempty"`
	Owner               string   `json:"owner,omitempty"`
	Description         string   `json:"description,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	HasCartographerYAML bool     `json:"has_cartographer_yaml"`
	WebURL              string   `json:"web_url"`
}

func (s *Server) handleListServices(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	cat := s.store.GetCatalog()

	if len(cat.Projects) == 0 {
		return mcp.NewToolResultText(emptyCacheMsg), nil
	}

	filterType := request.GetString("type", "")
	filterLifecycle := request.GetString("lifecycle", "")
	filterTag := request.GetString("tag", "")
	includeArchived := request.GetBool("include_archived", false)

	var services []serviceEntry
	for _, p := range cat.Projects {
		if p.Archived && !includeArchived {
			continue
		}

		entry := serviceEntry{
			Name:              p.Name,
			PathWithNamespace: p.PathWithNamespace,
			WebURL:            p.WebURL,
		}

		if p.Service != nil {
			entry.HasCartographerYAML = true
			entry.Name = p.Service.Name
			entry.Type = p.Service.Type
			entry.Lifecycle = p.Service.Lifecycle
			entry.Owner = p.Service.Owner
			entry.Description = p.Service.Description
			entry.Tags = p.Service.Tags
		}

		if !matchesFilters(entry, filterType, filterLifecycle, filterTag) {
			continue
		}

		services = append(services, entry)
	}

	resp := map[string]any{
		"cache_age_seconds": s.store.CacheAgeSeconds(),
		"refreshed_at":      cat.RefreshedAt,
		"total":             len(services),
		"services":          services,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func matchesFilters(
	entry serviceEntry,
	filterType, filterLifecycle, filterTag string,
) bool {
	if filterType != "" && !strings.EqualFold(entry.Type, filterType) {
		return false
	}
	if filterLifecycle != "" &&
		!strings.EqualFold(entry.Lifecycle, filterLifecycle) {
		return false
	}
	if filterTag != "" {
		found := false
		for _, t := range entry.Tags {
			if strings.EqualFold(t, filterTag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
