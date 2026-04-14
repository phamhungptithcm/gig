package workarea

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gig/internal/scm"
)

const envWorkareaFile = "GIG_WORKAREA_FILE"

type Definition struct {
	Name            string    `json:"name"`
	RepoTarget      string    `json:"repoTarget,omitempty"`
	Path            string    `json:"path,omitempty"`
	ConfigPath      string    `json:"configPath,omitempty"`
	FromBranch      string    `json:"fromBranch,omitempty"`
	ToBranch        string    `json:"toBranch,omitempty"`
	EnvironmentSpec string    `json:"environmentSpec,omitempty"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty"`
	LastUsedAt      time.Time `json:"lastUsedAt,omitempty"`
}

type RecentRepository struct {
	Provider   string    `json:"provider"`
	Root       string    `json:"root"`
	Name       string    `json:"name,omitempty"`
	SelectedAt time.Time `json:"selectedAt,omitempty"`
}

type State struct {
	Current            string             `json:"current,omitempty"`
	Workareas          []Definition       `json:"workareas,omitempty"`
	RecentRepositories []RecentRepository `json:"recentRepositories,omitempty"`
}

type Store struct {
	filePath  string
	cacheRoot string
	now       func() time.Time
}

func NewStore() (*Store, error) {
	if override := strings.TrimSpace(os.Getenv(envWorkareaFile)); override != "" {
		resolvedPath, err := filepath.Abs(override)
		if err != nil {
			return nil, err
		}
		return &Store{
			filePath:  resolvedPath,
			cacheRoot: filepath.Join(filepath.Dir(resolvedPath), "workareas"),
			now:       time.Now,
		}, nil
	}

	configRoot, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	gigRoot := filepath.Join(configRoot, "gig")
	return &Store{
		filePath:  filepath.Join(gigRoot, "workareas.json"),
		cacheRoot: filepath.Join(gigRoot, "workareas"),
		now:       time.Now,
	}, nil
}

func NewStoreAt(filePath string) *Store {
	resolvedPath := filePath
	if absolute, err := filepath.Abs(filePath); err == nil {
		resolvedPath = absolute
	}
	return &Store{
		filePath:  resolvedPath,
		cacheRoot: filepath.Join(filepath.Dir(resolvedPath), "workareas"),
		now:       time.Now,
	}
}

func (s *Store) FilePath() string {
	return s.filePath
}

func (s *Store) List() ([]Definition, string, error) {
	state, err := s.load()
	if err != nil {
		return nil, "", err
	}
	return append([]Definition(nil), state.Workareas...), state.Current, nil
}

func (s *Store) RecentWorkareas(limit int) ([]Definition, error) {
	state, err := s.load()
	if err != nil {
		return nil, err
	}

	workareas := append([]Definition(nil), state.Workareas...)
	sort.SliceStable(workareas, func(i, j int) bool {
		left := workareas[i].LastUsedAt
		right := workareas[j].LastUsedAt
		switch {
		case left.Equal(right):
			return strings.ToLower(workareas[i].Name) < strings.ToLower(workareas[j].Name)
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.After(right)
		}
	})
	if limit > 0 && len(workareas) > limit {
		workareas = workareas[:limit]
	}
	return workareas, nil
}

func (s *Store) Get(name string) (Definition, bool, error) {
	state, err := s.load()
	if err != nil {
		return Definition{}, false, err
	}
	index := findIndex(state.Workareas, name)
	if index < 0 {
		return Definition{}, false, nil
	}
	return state.Workareas[index], true, nil
}

func (s *Store) Current() (Definition, bool, error) {
	state, err := s.load()
	if err != nil {
		return Definition{}, false, err
	}
	if len(state.Workareas) == 0 {
		return Definition{}, false, nil
	}
	if strings.TrimSpace(state.Current) == "" && len(state.Workareas) == 1 {
		return state.Workareas[0], true, nil
	}
	index := findIndex(state.Workareas, state.Current)
	if index < 0 {
		return Definition{}, false, nil
	}
	return state.Workareas[index], true, nil
}

func (s *Store) Save(def Definition, setCurrent bool) (Definition, error) {
	normalized, err := normalizeDefinition(def)
	if err != nil {
		return Definition{}, err
	}

	state, err := s.load()
	if err != nil {
		return Definition{}, err
	}

	now := s.now().UTC()
	index := findIndex(state.Workareas, normalized.Name)
	if index >= 0 {
		normalized.CreatedAt = state.Workareas[index].CreatedAt
	} else {
		normalized.CreatedAt = now
	}
	normalized.UpdatedAt = now
	if setCurrent {
		normalized.LastUsedAt = now
	}

	if index >= 0 {
		state.Workareas[index] = normalized
	} else {
		state.Workareas = append(state.Workareas, normalized)
	}
	sortDefinitions(state.Workareas)
	if setCurrent || len(state.Workareas) == 1 || strings.TrimSpace(state.Current) == "" {
		state.Current = normalized.Name
	}
	if err := s.save(state); err != nil {
		return Definition{}, err
	}
	if err := os.MkdirAll(s.ScopePath(normalized), 0o755); err != nil {
		return Definition{}, err
	}
	return normalized, nil
}

func (s *Store) Use(name string) (Definition, error) {
	state, err := s.load()
	if err != nil {
		return Definition{}, err
	}
	index := findIndex(state.Workareas, name)
	if index < 0 {
		return Definition{}, fmt.Errorf("workarea %q was not found", strings.TrimSpace(name))
	}

	now := s.now().UTC()
	state.Workareas[index].LastUsedAt = now
	state.Workareas[index].UpdatedAt = now
	state.Current = state.Workareas[index].Name
	if err := s.save(state); err != nil {
		return Definition{}, err
	}
	if err := os.MkdirAll(s.ScopePath(state.Workareas[index]), 0o755); err != nil {
		return Definition{}, err
	}
	return state.Workareas[index], nil
}

func (s *Store) Touch(name string) (Definition, error) {
	state, err := s.load()
	if err != nil {
		return Definition{}, err
	}
	index := findIndex(state.Workareas, name)
	if index < 0 {
		return Definition{}, fmt.Errorf("workarea %q was not found", strings.TrimSpace(name))
	}
	now := s.now().UTC()
	state.Workareas[index].LastUsedAt = now
	state.Workareas[index].UpdatedAt = now
	if err := s.save(state); err != nil {
		return Definition{}, err
	}
	return state.Workareas[index], nil
}

func (s *Store) SaveInferredDefaults(name, fromBranch, toBranch, environmentSpec string) (Definition, bool, error) {
	state, err := s.load()
	if err != nil {
		return Definition{}, false, err
	}

	index := findIndex(state.Workareas, name)
	if index < 0 {
		return Definition{}, false, fmt.Errorf("workarea %q was not found", strings.TrimSpace(name))
	}

	definition := state.Workareas[index]
	changed := false

	if definition.FromBranch == "" && strings.TrimSpace(fromBranch) != "" {
		definition.FromBranch = strings.TrimSpace(fromBranch)
		changed = true
	}
	if definition.ToBranch == "" && strings.TrimSpace(toBranch) != "" {
		definition.ToBranch = strings.TrimSpace(toBranch)
		changed = true
	}
	if definition.EnvironmentSpec == "" && strings.TrimSpace(environmentSpec) != "" {
		definition.EnvironmentSpec = strings.TrimSpace(environmentSpec)
		changed = true
	}
	if !changed {
		return definition, false, nil
	}

	definition.UpdatedAt = s.now().UTC()
	state.Workareas[index] = definition
	if err := s.save(state); err != nil {
		return Definition{}, false, err
	}
	return definition, true, nil
}

func (s *Store) EnsureRemoteRepository(repository scm.Repository) (Definition, bool, error) {
	if strings.TrimSpace(repository.Root) == "" || !repository.Type.IsRemote() {
		return Definition{}, false, fmt.Errorf("a remote repository target is required")
	}

	state, err := s.load()
	if err != nil {
		return Definition{}, false, err
	}

	now := s.now().UTC()
	for index, definition := range state.Workareas {
		if strings.EqualFold(strings.TrimSpace(definition.RepoTarget), strings.TrimSpace(repository.Root)) {
			state.Workareas[index].LastUsedAt = now
			state.Workareas[index].UpdatedAt = now
			state.Current = state.Workareas[index].Name
			if err := s.save(state); err != nil {
				return Definition{}, false, err
			}
			if err := os.MkdirAll(s.ScopePath(state.Workareas[index]), 0o755); err != nil {
				return Definition{}, false, err
			}
			return state.Workareas[index], false, nil
		}
	}

	name := uniqueRepositoryWorkareaName(state.Workareas, repository)
	definition := Definition{
		Name:       name,
		RepoTarget: strings.TrimSpace(repository.Root),
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	state.Workareas = append(state.Workareas, definition)
	sortDefinitions(state.Workareas)
	state.Current = definition.Name
	if err := s.save(state); err != nil {
		return Definition{}, false, err
	}
	if err := os.MkdirAll(s.ScopePath(definition), 0o755); err != nil {
		return Definition{}, false, err
	}
	return definition, true, nil
}

func (s *Store) ScopePath(def Definition) string {
	if strings.TrimSpace(def.Path) != "" {
		return def.Path
	}
	return filepath.Join(s.cacheRoot, slugify(def.Name))
}

func (s *Store) RecordRepositorySelection(repository scm.Repository) error {
	if strings.TrimSpace(repository.Root) == "" || !repository.Type.IsRemote() {
		return nil
	}

	state, err := s.load()
	if err != nil {
		return err
	}

	now := s.now().UTC()
	record := RecentRepository{
		Provider:   string(repository.Type),
		Root:       strings.TrimSpace(repository.Root),
		Name:       strings.TrimSpace(repository.Name),
		SelectedAt: now,
	}

	index := -1
	for i, candidate := range state.RecentRepositories {
		if strings.EqualFold(strings.TrimSpace(candidate.Root), record.Root) {
			index = i
			break
		}
	}
	if index >= 0 {
		state.RecentRepositories[index] = record
	} else {
		state.RecentRepositories = append(state.RecentRepositories, record)
	}
	sortRecentRepositories(state.RecentRepositories)
	if len(state.RecentRepositories) > 25 {
		state.RecentRepositories = state.RecentRepositories[:25]
	}
	return s.save(state)
}

func (s *Store) RecentRepositories(provider scm.Type, limit int) ([]RecentRepository, error) {
	state, err := s.load()
	if err != nil {
		return nil, err
	}

	providerName := strings.TrimSpace(string(provider))
	results := make([]RecentRepository, 0, len(state.RecentRepositories))
	for _, repository := range state.RecentRepositories {
		if providerName != "" && !strings.EqualFold(strings.TrimSpace(repository.Provider), providerName) {
			continue
		}
		results = append(results, repository)
	}
	sortRecentRepositories(results)
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Store) load() (State, error) {
	content, err := os.ReadFile(s.filePath)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return State{}, nil
	default:
		return State{}, err
	}

	var state State
	if err := json.Unmarshal(content, &state); err != nil {
		return State{}, fmt.Errorf("parse workareas %s: %w", s.filePath, err)
	}
	sortDefinitions(state.Workareas)
	return state, nil
}

func (s *Store) save(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(s.filePath, payload, 0o644)
}

func normalizeDefinition(def Definition) (Definition, error) {
	normalized := Definition{
		Name:            strings.TrimSpace(def.Name),
		RepoTarget:      strings.TrimSpace(def.RepoTarget),
		Path:            strings.TrimSpace(def.Path),
		ConfigPath:      strings.TrimSpace(def.ConfigPath),
		FromBranch:      strings.TrimSpace(def.FromBranch),
		ToBranch:        strings.TrimSpace(def.ToBranch),
		EnvironmentSpec: strings.TrimSpace(def.EnvironmentSpec),
	}
	if normalized.Name == "" {
		return Definition{}, fmt.Errorf("workarea name is required")
	}
	if normalized.RepoTarget == "" && normalized.Path == "" {
		return Definition{}, fmt.Errorf("workarea %q must set either a repo target or a path", normalized.Name)
	}
	return normalized, nil
}

func findIndex(definitions []Definition, name string) int {
	name = strings.TrimSpace(name)
	for index, definition := range definitions {
		if strings.EqualFold(definition.Name, name) {
			return index
		}
	}
	return -1
}

func sortDefinitions(definitions []Definition) {
	sort.Slice(definitions, func(i, j int) bool {
		left := strings.ToLower(definitions[i].Name)
		right := strings.ToLower(definitions[j].Name)
		if left == right {
			return definitions[i].Name < definitions[j].Name
		}
		return left < right
	})
}

func sortRecentRepositories(repositories []RecentRepository) {
	sort.SliceStable(repositories, func(i, j int) bool {
		left := repositories[i].SelectedAt
		right := repositories[j].SelectedAt
		switch {
		case left.Equal(right):
			return strings.ToLower(repositories[i].Root) < strings.ToLower(repositories[j].Root)
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.After(right)
		}
	})
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "default"
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "default"
	}
	return slug
}

func uniqueRepositoryWorkareaName(definitions []Definition, repository scm.Repository) string {
	base := strings.TrimSpace(repository.Name)
	if base == "" {
		base = strings.TrimSpace(repository.Root)
	}
	base = slugify(base)
	if !hasDefinitionName(definitions, base) {
		return base
	}

	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if !hasDefinitionName(definitions, candidate) {
			return candidate
		}
	}
}

func hasDefinitionName(definitions []Definition, name string) bool {
	return findIndex(definitions, name) >= 0
}
