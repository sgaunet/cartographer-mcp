// Package graph provides dependency graph construction and traversal.
package graph

import "github.com/sgaunet/cartographer-mcp/internal/models"

// ReverseEdge represents who depends on a service.
type ReverseEdge struct {
	FromService string `json:"from_service"`
	Type        string `json:"type,omitempty"`
}

// Graph holds forward and reverse adjacency maps built from cached projects.
type Graph struct {
	forward map[string][]models.Dependency
	reverse map[string][]ReverseEdge
}

// BuildGraph constructs forward and reverse adjacency maps from cached projects.
func BuildGraph(projects []models.Project) *Graph {
	g := &Graph{
		forward: make(map[string][]models.Dependency),
		reverse: make(map[string][]ReverseEdge),
	}

	for _, p := range projects {
		if p.Service == nil {
			continue
		}
		name := p.Service.Name
		for _, dep := range p.Service.Dependencies {
			g.forward[name] = append(g.forward[name], dep)
			g.reverse[dep.Service] = append(g.reverse[dep.Service], ReverseEdge{
				FromService: name,
				Type:        dep.Type,
			})
		}
	}

	return g
}

// ForwardDeps returns direct dependencies for a service.
func (g *Graph) ForwardDeps(service string) []models.Dependency {
	return g.forward[service]
}

// ReverseDeps returns direct dependents of a service.
func (g *Graph) ReverseDeps(service string) []ReverseEdge {
	return g.reverse[service]
}

const maxDepth = 50

// TransitiveDeps returns all transitive forward dependencies using BFS.
func (g *Graph) TransitiveDeps(service string) ([]models.Dependency, bool) {
	visited := make(map[string]bool)
	var result []models.Dependency
	hasCycle := false

	queue := []string{service}
	visited[service] = true
	depth := 0

	for len(queue) > 0 && depth < maxDepth {
		nextQueue := []string{}
		for _, current := range queue {
			for _, dep := range g.forward[current] {
				if dep.Service == service {
					hasCycle = true
					continue
				}
				if !visited[dep.Service] {
					visited[dep.Service] = true
					result = append(result, dep)
					nextQueue = append(nextQueue, dep.Service)
				}
			}
		}
		queue = nextQueue
		depth++
	}

	return result, hasCycle
}

// TransitiveDependents returns all transitive reverse dependencies using BFS.
func (g *Graph) TransitiveDependents(service string) ([]ReverseEdge, bool) {
	visited := make(map[string]bool)
	var result []ReverseEdge
	hasCycle := false

	queue := []string{service}
	visited[service] = true
	depth := 0

	for len(queue) > 0 && depth < maxDepth {
		nextQueue := []string{}
		for _, current := range queue {
			for _, edge := range g.reverse[current] {
				if edge.FromService == service {
					hasCycle = true
					continue
				}
				if !visited[edge.FromService] {
					visited[edge.FromService] = true
					result = append(result, edge)
					nextQueue = append(nextQueue, edge.FromService)
				}
			}
		}
		queue = nextQueue
		depth++
	}

	return result, hasCycle
}

// ServiceExists checks if a service is known in the graph.
func (g *Graph) ServiceExists(service string) bool {
	_, hasFwd := g.forward[service]
	_, hasRev := g.reverse[service]
	return hasFwd || hasRev
}

// AllServiceNames returns all service names known in the graph.
func (g *Graph) AllServiceNames() []string {
	seen := make(map[string]bool)
	for k := range g.forward {
		seen[k] = true
	}
	for k := range g.reverse {
		seen[k] = true
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}
