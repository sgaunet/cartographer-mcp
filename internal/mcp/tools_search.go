package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sgaunet/cartographer-mcp/internal/models"
)

func (s *Server) registerSearchServices() {
	tool := mcp.NewTool("search_services",
		mcp.WithDescription(
			"Search services by keyword across names, descriptions, tags, and outputs",
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search term"),
		),
	)

	s.addTool(tool, s.handleSearchServices)
}

type searchResult struct {
	Name        string `json:"name"`
	MatchReason string `json:"match_reason"`
	Type        string `json:"type,omitempty"`
	Lifecycle   string `json:"lifecycle,omitempty"`
}

func (s *Server) handleSearchServices(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cat := s.store.GetCatalog()
	queryLower := strings.ToLower(query)

	var results []searchResult

	for _, p := range cat.Projects {
		if p.Service == nil {
			if reason := matchProject(p, queryLower); reason != "" {
				results = append(results, searchResult{
					Name:        p.Name,
					MatchReason: reason,
				})
			}
			continue
		}

		if reason := matchService(p, queryLower); reason != "" {
			results = append(results, searchResult{
				Name:        p.Service.Name,
				MatchReason: reason,
				Type:        p.Service.Type,
				Lifecycle:   p.Service.Lifecycle,
			})
		}
	}

	resp := map[string]any{
		"cache_age_seconds": s.store.CacheAgeSeconds(),
		"refreshed_at":      cat.RefreshedAt,
		"query":             query,
		"results":           results,
		"total":             len(results),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func matchProject(p models.Project, query string) string {
	if strings.Contains(strings.ToLower(p.Name), query) {
		return "project name"
	}
	if strings.Contains(strings.ToLower(p.Description), query) {
		return "project description"
	}
	return ""
}

func matchService(p models.Project, query string) string {
	svc := p.Service

	if strings.Contains(strings.ToLower(svc.Name), query) {
		return "service name"
	}
	if strings.Contains(strings.ToLower(svc.Description), query) {
		return "service description"
	}
	for _, tag := range svc.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return "tag: " + tag
		}
	}
	for _, out := range svc.Outputs {
		if strings.Contains(strings.ToLower(out.Name), query) {
			return "output: " + out.Name + " (type: " + out.Type + ")"
		}
		if strings.Contains(strings.ToLower(out.Type), query) {
			return "output type: " + out.Type + " (" + out.Name + ")"
		}
	}
	if strings.Contains(strings.ToLower(p.Description), query) {
		return "project description"
	}
	return ""
}
