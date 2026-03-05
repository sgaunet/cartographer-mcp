package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerGetService() {
	tool := mcp.NewTool("get_service",
		mcp.WithDescription(
			"Get full details for a specific service by name or path",
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Service name or project path"),
		),
	)

	s.addTool(tool, s.handleGetService)
}

func (s *Server) handleGetService(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cat := s.store.GetCatalog()
	project := findProjectByName(cat, name)

	if project == nil {
		suggestions := fuzzyMatch(cat, name)
		resp := map[string]any{
			"cache_age_seconds": s.store.CacheAgeSeconds(),
			"refreshed_at":      cat.RefreshedAt,
			"found":             false,
			"suggestions":       suggestions,
		}
		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return mcp.NewToolResultError("failed to marshal"), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	resp := map[string]any{
		"cache_age_seconds": s.store.CacheAgeSeconds(),
		"refreshed_at":      cat.RefreshedAt,
		"found":             true,
		"project": map[string]any{
			"id":                  project.ID,
			"name":                project.Name,
			"path_with_namespace": project.PathWithNamespace,
			"description":         project.Description,
			"default_branch":      project.DefaultBranch,
			"web_url":             project.WebURL,
			"archived":            project.Archived,
			"last_activity_at":    project.LastActivityAt,
			"latest_version":      project.LatestVersion,
			"readme_content":      project.ReadmeContent,
			"fetched_at":          project.FetchedAt,
		},
	}

	if project.Service != nil {
		resp["service"] = project.Service
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
