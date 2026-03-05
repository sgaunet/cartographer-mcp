// Package schema handles .cartographer.yaml parsing and validation.
package schema

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sgaunet/cartographer-mcp/internal/models"
)

var (
	validServiceTypes = map[string]bool{
		"api": true, "worker": true, "library": true,
		"frontend": true, "cli": true, "other": true,
	}
	validLifecycles = map[string]bool{
		"development": true, "staging": true, "production": true,
		"deprecated": true, "archived": true,
	}
	validDepTypes = map[string]bool{
		"api": true, "events": true, "database": true,
		"library": true, "grpc": true,
	}
	validOutputTypes = map[string]bool{
		"api": true, "events": true, "library": true,
		"artifacts": true, "email": true, "webhook": true,
	}
	knownTopLevelKeys = map[string]bool{
		"schema_version": true, "service": true,
		"dependencies": true, "outputs": true,
	}
)

// cartographerFile is the raw YAML structure for .cartographer.yaml.
type cartographerFile struct {
	SchemaVersion int `yaml:"schema_version"`
	Service       struct {
		Name        string   `yaml:"name"`
		Type        string   `yaml:"type"`
		Lifecycle   string   `yaml:"lifecycle"`
		Owner       string   `yaml:"owner"`
		Description string   `yaml:"description"`
		Tags        []string `yaml:"tags"`
	} `yaml:"service"`
	Dependencies []models.Dependency `yaml:"dependencies"`
	Outputs      []models.Output     `yaml:"outputs"`
}

// ParseCartographerYAML parses and validates a .cartographer.yaml file.
// Returns an error only if the YAML cannot be parsed at all.
func ParseCartographerYAML(data []byte) (*models.ServiceMetadata, error) {
	var file cartographerFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	var rawMap map[string]any
	_ = yaml.Unmarshal(data, &rawMap)

	sm := &models.ServiceMetadata{
		SchemaVersion: file.SchemaVersion,
		Name:          file.Service.Name,
		Type:          file.Service.Type,
		Lifecycle:     file.Service.Lifecycle,
		Owner:         file.Service.Owner,
		Description:   file.Service.Description,
		Tags:          file.Service.Tags,
		Dependencies:  file.Dependencies,
		Outputs:       file.Outputs,
	}

	sm.ValidationWarnings = validate(sm, rawMap)
	return sm, nil
}

func validate(sm *models.ServiceMetadata, rawMap map[string]any) []string {
	var w []string
	w = validateCore(w, sm)
	w = validateDeps(w, sm)
	w = validateOutputs(w, sm)
	w = validateUnknownKeys(w, rawMap)
	return w
}

func validateCore(w []string, sm *models.ServiceMetadata) []string {
	if sm.SchemaVersion != 1 {
		w = append(w, fmt.Sprintf(
			"schema_version must be 1, got %d", sm.SchemaVersion))
	}
	if strings.TrimSpace(sm.Name) == "" {
		w = append(w, "service.name is required and must be non-empty")
	}
	if !validServiceTypes[sm.Type] {
		w = append(w, fmt.Sprintf(
			"service.type invalid, got %q", sm.Type))
	}
	if !validLifecycles[sm.Lifecycle] {
		w = append(w, fmt.Sprintf(
			"service.lifecycle invalid, got %q", sm.Lifecycle))
	}
	return w
}

func validateDeps(w []string, sm *models.ServiceMetadata) []string {
	for i, dep := range sm.Dependencies {
		if strings.TrimSpace(dep.Service) == "" {
			w = append(w, fmt.Sprintf(
				"dependencies[%d].service must be non-empty", i))
		}
		if dep.Type != "" && !validDepTypes[dep.Type] {
			w = append(w, fmt.Sprintf(
				"dependencies[%d].type %q is not recognized", i, dep.Type))
		}
	}
	return w
}

func validateOutputs(w []string, sm *models.ServiceMetadata) []string {
	for i, out := range sm.Outputs {
		if strings.TrimSpace(out.Name) == "" {
			w = append(w, fmt.Sprintf(
				"outputs[%d].name must be non-empty", i))
		}
		if !validOutputTypes[out.Type] {
			w = append(w, fmt.Sprintf(
				"outputs[%d].type invalid, got %q", i, out.Type))
		}
	}
	return w
}

func validateUnknownKeys(w []string, rawMap map[string]any) []string {
	for key := range rawMap {
		if !knownTopLevelKeys[key] {
			w = append(w, fmt.Sprintf("unknown top-level key %q", key))
		}
	}
	return w
}
