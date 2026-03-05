package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sgaunet/cartographer-mcp/internal/graph"
)

func (s *Server) registerGetDependencies() {
	tool := mcp.NewTool("get_dependencies",
		mcp.WithDescription(
			"Get forward dependencies of a service (what it depends on)",
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Service name"),
		),
		mcp.WithBoolean("transitive",
			mcp.Description("Include transitive deps (default: false)"),
		),
	)

	s.addTool(tool, s.handleGetDependencies)
}

func (s *Server) registerGetDependents() {
	tool := mcp.NewTool("get_dependents",
		mcp.WithDescription(
			"Get reverse dependencies (what depends on this service)",
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Service name"),
		),
		mcp.WithBoolean("transitive",
			mcp.Description("Include transitive dependents (default: false)"),
		),
	)

	s.addTool(tool, s.handleGetDependents)
}

type forwardDepEntry struct {
	Service   string `json:"service"`
	Type      string `json:"type,omitempty"`
	Resolved  bool   `json:"resolved"`
	Lifecycle string `json:"lifecycle,omitempty"`
}

type reverseDepEntry struct {
	Service   string `json:"service"`
	Type      string `json:"type,omitempty"`
	Lifecycle string `json:"lifecycle,omitempty"`
}

func (s *Server) handleGetDependencies(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	transitive := request.GetBool("transitive", false)

	cat := s.store.GetCatalog()
	g := graph.BuildGraph(cat.Projects)

	if !serviceExistsInCatalog(cat, name) {
		return mcp.NewToolResultError("service not found: " + name), nil
	}

	var deps []forwardDepEntry
	unresolvedCount := 0

	if transitive {
		fwdDeps, _ := g.TransitiveDeps(name)
		for _, d := range fwdDeps {
			entry := forwardDepEntry{Service: d.Service, Type: d.Type}
			entry.Resolved, entry.Lifecycle = resolveService(cat, d.Service)
			if !entry.Resolved {
				unresolvedCount++
			}
			deps = append(deps, entry)
		}
	} else {
		for _, d := range g.ForwardDeps(name) {
			entry := forwardDepEntry{Service: d.Service, Type: d.Type}
			entry.Resolved, entry.Lifecycle = resolveService(cat, d.Service)
			if !entry.Resolved {
				unresolvedCount++
			}
			deps = append(deps, entry)
		}
	}

	resp := map[string]any{
		"cache_age_seconds": s.store.CacheAgeSeconds(),
		"refreshed_at":      cat.RefreshedAt,
		"service":           name,
		"transitive":        transitive,
		"dependencies":      deps,
		"unresolved_count":  unresolvedCount,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleGetDependents(
	_ context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	transitive := request.GetBool("transitive", false)

	cat := s.store.GetCatalog()
	g := graph.BuildGraph(cat.Projects)

	if !serviceExistsInCatalog(cat, name) {
		return mcp.NewToolResultError("service not found: " + name), nil
	}

	var dependents []reverseDepEntry

	if transitive {
		revDeps, _ := g.TransitiveDependents(name)
		for _, d := range revDeps {
			entry := reverseDepEntry{Service: d.FromService, Type: d.Type}
			_, entry.Lifecycle = resolveService(cat, d.FromService)
			dependents = append(dependents, entry)
		}
	} else {
		for _, d := range g.ReverseDeps(name) {
			entry := reverseDepEntry{Service: d.FromService, Type: d.Type}
			_, entry.Lifecycle = resolveService(cat, d.FromService)
			dependents = append(dependents, entry)
		}
	}

	resp := map[string]any{
		"cache_age_seconds": s.store.CacheAgeSeconds(),
		"refreshed_at":      cat.RefreshedAt,
		"service":           name,
		"transitive":        transitive,
		"dependents":        dependents,
		"total":             len(dependents),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
