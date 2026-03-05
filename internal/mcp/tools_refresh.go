package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerRefreshCache() {
	tool := mcp.NewTool("refresh_cache",
		mcp.WithDescription(
			"Trigger a full cache refresh from GitLab",
		),
		mcp.WithString("groups",
			mcp.Description(
				"Comma-separated GitLab group paths (overrides config)",
			),
		),
	)

	s.addTool(tool, s.handleRefreshCache)
}

func (s *Server) handleRefreshCache(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	groups := s.config.Groups

	if groupsStr := request.GetString("groups", ""); groupsStr != "" {
		groups = nil
		for g := range strings.SplitSeq(groupsStr, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				groups = append(groups, g)
			}
		}
	}

	if len(groups) == 0 {
		return mcp.NewToolResultError(
			"no groups configured. Set CARTOGRAPHER_GROUPS or pass groups parameter.",
		), nil
	}

	if s.refresher == nil {
		return mcp.NewToolResultError(
			"refresher not initialized (missing GITLAB_TOKEN?)",
		), nil
	}

	result, err := s.refresher.Refresh(ctx, s.config.GitLabURI, groups)
	if err != nil {
		return mcp.NewToolResultError("refresh failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"status":                     "completed",
		"refreshed_at":               result.RefreshedAt,
		"duration_ms":                result.DurationMs,
		"projects_discovered":        result.ProjectsDiscovered,
		"projects_with_cartographer": result.ProjectsWithCartographer,
		"diagnostics":               result.Diagnostics,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
