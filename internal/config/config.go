package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultTicketPattern = `\b[A-Z][A-Z0-9]+-\d+\b`

var defaultEnvironments = []Environment{
	{Name: "dev", Branch: "dev"},
	{Name: "test", Branch: "test"},
	{Name: "prod", Branch: "main"},
}

var configFileNames = []string{
	"gig.yaml",
	"gig.yml",
	".gig.yaml",
	".gig.yml",
}

type Environment struct {
	Name   string `yaml:"name" json:"name"`
	Branch string `yaml:"branch" json:"branch"`
}

type Repository struct {
	Name    string   `yaml:"name" json:"name,omitempty"`
	Path    string   `yaml:"path" json:"path,omitempty"`
	Service string   `yaml:"service" json:"service,omitempty"`
	Owner   string   `yaml:"owner" json:"owner,omitempty"`
	Kind    string   `yaml:"kind" json:"kind,omitempty"`
	Notes   []string `yaml:"notes" json:"notes,omitempty"`
}

type Config struct {
	TicketPattern string        `yaml:"ticketPattern" json:"ticketPattern,omitempty"`
	Environments  []Environment `yaml:"environments" json:"environments,omitempty"`
	Repositories  []Repository  `yaml:"repositories" json:"repositories,omitempty"`
}

type Loaded struct {
	Path                  string `json:"path,omitempty"`
	Found                 bool   `json:"found"`
	ExplicitEnvironments  bool   `json:"explicitEnvironments"`
	ExplicitRepositories  bool   `json:"explicitRepositories"`
	ExplicitTicketPattern bool   `json:"explicitTicketPattern"`
	Config                Config `json:"config"`
}

func Default() Config {
	return Config{
		TicketPattern: defaultTicketPattern,
		Environments:  append([]Environment(nil), defaultEnvironments...),
	}
}

func LoadForPath(path, explicitPath string) (Loaded, error) {
	base := Default()

	configPath, found, err := locate(path, explicitPath)
	if err != nil {
		return Loaded{}, err
	}
	if !found {
		return Loaded{
			Config: base,
		}, nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read config: %w", err)
	}

	var fileConfig Config
	if err := yaml.Unmarshal(content, &fileConfig); err != nil {
		return Loaded{}, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	loaded := Loaded{
		Path:                  configPath,
		Found:                 true,
		ExplicitEnvironments:  len(fileConfig.Environments) > 0,
		ExplicitRepositories:  len(fileConfig.Repositories) > 0,
		ExplicitTicketPattern: strings.TrimSpace(fileConfig.TicketPattern) != "",
		Config:                merge(base, fileConfig),
	}

	return loaded, nil
}

func (c Config) FindRepository(workspacePath, repoRoot, repoName string) (Repository, bool) {
	relativePath := normalizeRelativePath(workspacePath, repoRoot)
	absolutePath := normalizeConfigPath(repoRoot)

	for _, repository := range c.Repositories {
		switch {
		case strings.TrimSpace(repository.Path) != "":
			candidate := normalizeConfigPath(repository.Path)
			if candidate == relativePath || candidate == absolutePath {
				return repository, true
			}
		case strings.TrimSpace(repository.Name) != "":
			if strings.EqualFold(strings.TrimSpace(repository.Name), strings.TrimSpace(repoName)) {
				return repository, true
			}
		}
	}

	return Repository{}, false
}

func locate(path, explicitPath string) (string, bool, error) {
	if strings.TrimSpace(explicitPath) != "" {
		resolvedPath, err := filepath.Abs(explicitPath)
		if err != nil {
			return "", false, err
		}
		if _, err := os.Stat(resolvedPath); err != nil {
			return "", false, fmt.Errorf("config file %s: %w", resolvedPath, err)
		}
		return resolvedPath, true, nil
	}

	start, err := normalizeSearchPath(path)
	if err != nil {
		return "", false, err
	}

	for current := start; ; current = filepath.Dir(current) {
		for _, name := range configFileNames {
			candidate := filepath.Join(current, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, true, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}

	return "", false, nil
}

func merge(base, override Config) Config {
	merged := base

	if ticketPattern := strings.TrimSpace(override.TicketPattern); ticketPattern != "" {
		merged.TicketPattern = ticketPattern
	}
	if len(override.Environments) > 0 {
		merged.Environments = normalizeEnvironments(override.Environments)
	}
	if len(override.Repositories) > 0 {
		merged.Repositories = normalizeRepositories(override.Repositories)
	}

	return merged
}

func normalizeEnvironments(environments []Environment) []Environment {
	normalized := make([]Environment, 0, len(environments))
	seen := map[string]struct{}{}

	for _, environment := range environments {
		name := strings.TrimSpace(environment.Name)
		branch := strings.TrimSpace(environment.Branch)
		if name == "" || branch == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, Environment{Name: name, Branch: branch})
	}

	return normalized
}

func normalizeRepositories(repositories []Repository) []Repository {
	normalized := make([]Repository, 0, len(repositories))
	for _, repository := range repositories {
		entry := Repository{
			Name:    strings.TrimSpace(repository.Name),
			Path:    normalizeConfigPath(repository.Path),
			Service: strings.TrimSpace(repository.Service),
			Owner:   strings.TrimSpace(repository.Owner),
			Kind:    strings.TrimSpace(repository.Kind),
			Notes:   normalizeNotes(repository.Notes),
		}
		if entry.Name == "" && entry.Path == "" {
			continue
		}
		normalized = append(normalized, entry)
	}

	sort.Slice(normalized, func(i, j int) bool {
		left := normalized[i].Path
		if left == "" {
			left = normalized[i].Name
		}
		right := normalized[j].Path
		if right == "" {
			right = normalized[j].Name
		}
		return left < right
	})

	return normalized
}

func normalizeNotes(notes []string) []string {
	normalized := make([]string, 0, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		normalized = append(normalized, note)
	}
	return normalized
}

func normalizeSearchPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}

	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return resolvedPath, nil
	}
	return filepath.Dir(resolvedPath), nil
}

func normalizeRelativePath(workspacePath, repoRoot string) string {
	relativePath, err := filepath.Rel(workspacePath, repoRoot)
	if err != nil {
		return normalizeConfigPath(repoRoot)
	}

	return normalizeConfigPath(relativePath)
}

func normalizeConfigPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}
