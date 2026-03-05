package mcp

import (
	"strings"

	"github.com/sgaunet/cartographer-mcp/internal/models"
)

const (
	maxSuggestions       = 5
	levenshteinThreshold = 3
)

// serviceExistsInCatalog checks if a service name exists in the catalog.
func serviceExistsInCatalog(cat *models.Catalog, name string) bool {
	for _, p := range cat.Projects {
		if p.Service != nil && strings.EqualFold(p.Service.Name, name) {
			return true
		}
		if strings.EqualFold(p.PathWithNamespace, name) {
			return true
		}
	}
	return false
}

// resolveService checks if a service name can be resolved in the catalog.
func resolveService(cat *models.Catalog, name string) (bool, string) {
	for _, p := range cat.Projects {
		if p.Service != nil && strings.EqualFold(p.Service.Name, name) {
			return true, p.Service.Lifecycle
		}
	}
	return false, ""
}

// findProjectByName finds a project by service name or path.
func findProjectByName(cat *models.Catalog, name string) *models.Project {
	for i, p := range cat.Projects {
		if p.Service != nil && strings.EqualFold(p.Service.Name, name) {
			return &cat.Projects[i]
		}
	}
	for i, p := range cat.Projects {
		if strings.EqualFold(p.PathWithNamespace, name) {
			return &cat.Projects[i]
		}
	}
	return nil
}

// fuzzyMatch returns service names that are close to the query.
func fuzzyMatch(cat *models.Catalog, query string) []string {
	query = strings.ToLower(query)
	var suggestions []string

	for _, p := range cat.Projects {
		name := projectDisplayName(p)
		if strings.Contains(strings.ToLower(name), query) ||
			strings.Contains(strings.ToLower(p.PathWithNamespace), query) {
			suggestions = append(suggestions, name)
		}
	}

	if len(suggestions) == 0 {
		for _, p := range cat.Projects {
			name := projectDisplayName(p)
			if levenshteinClose(strings.ToLower(name), query) {
				suggestions = append(suggestions, name)
			}
		}
	}

	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}
	return suggestions
}

func projectDisplayName(p models.Project) string {
	if p.Service != nil {
		return p.Service.Name
	}
	return p.Name
}

func levenshteinClose(a, b string) bool {
	if abs(len(a)-len(b)) > levenshteinThreshold {
		return false
	}
	return levenshtein(a, b) <= levenshteinThreshold
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
