package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const envAssistSessionFile = "GIG_ASSIST_SESSION_FILE"
const envWorkareaFile = "GIG_WORKAREA_FILE"

type Store struct {
	filePath string
	now      func() time.Time
}

func NewStore() (*Store, error) {
	if override := strings.TrimSpace(os.Getenv(envAssistSessionFile)); override != "" {
		resolvedPath, err := filepath.Abs(override)
		if err != nil {
			return nil, err
		}
		return &Store{filePath: resolvedPath, now: time.Now}, nil
	}

	if workareaFile := strings.TrimSpace(os.Getenv(envWorkareaFile)); workareaFile != "" {
		resolvedPath, err := filepath.Abs(filepath.Join(filepath.Dir(workareaFile), "assist-sessions.json"))
		if err != nil {
			return nil, err
		}
		return &Store{filePath: resolvedPath, now: time.Now}, nil
	}

	configRoot, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{
		filePath: filepath.Join(configRoot, "gig", "assist-sessions.json"),
		now:      time.Now,
	}, nil
}

func NewStoreAt(filePath string) *Store {
	resolvedPath := filePath
	if absolute, err := filepath.Abs(filePath); err == nil {
		resolvedPath = absolute
	}
	return &Store{filePath: resolvedPath, now: time.Now}
}

func (s *Store) FilePath() string {
	return s.filePath
}

func (s *Store) Current() (Session, bool, error) {
	state, err := s.load()
	if err != nil {
		return Session{}, false, err
	}
	return currentSessionFromState(state)
}

func (s *Store) CurrentForScope(workareaName, repoTarget, workspacePath string) (Session, bool, error) {
	state, err := s.load()
	if err != nil {
		return Session{}, false, err
	}
	if session, ok := findCurrentSessionForScope(state.Sessions, workareaName, repoTarget, workspacePath); ok {
		return session, true, nil
	}
	return Session{}, false, nil
}

func (s *Store) List() ([]Session, string, error) {
	state, err := s.load()
	if err != nil {
		return nil, "", err
	}
	return append([]Session(nil), state.Sessions...), state.Current, nil
}

func (s *Store) SaveCurrent(session Session) (Session, error) {
	normalized, err := normalizeSession(session)
	if err != nil {
		return Session{}, err
	}

	state, err := s.load()
	if err != nil {
		return Session{}, err
	}

	now := s.now().UTC()
	index := findIndex(state.Sessions, normalized.ID)
	if index >= 0 {
		normalized.CreatedAt = state.Sessions[index].CreatedAt
	} else {
		normalized.CreatedAt = now
	}
	normalized.UpdatedAt = now

	if index >= 0 {
		state.Sessions[index] = normalized
	} else {
		state.Sessions = append(state.Sessions, normalized)
	}
	sortSessions(state.Sessions)
	state.Current = normalized.ID

	if err := s.save(state); err != nil {
		return Session{}, err
	}
	return normalized, nil
}

func (s *Store) Touch(id, question, response, threadID string) (Session, error) {
	state, err := s.load()
	if err != nil {
		return Session{}, err
	}

	index := findIndex(state.Sessions, id)
	if index < 0 {
		return Session{}, fmt.Errorf("assist session %q was not found", strings.TrimSpace(id))
	}

	session := state.Sessions[index]
	if strings.TrimSpace(question) != "" {
		session.LastQuestion = strings.TrimSpace(question)
	}
	if strings.TrimSpace(response) != "" {
		session.LastResponse = strings.TrimSpace(response)
	}
	if strings.TrimSpace(threadID) != "" {
		session.ThreadID = strings.TrimSpace(threadID)
	}
	session.UpdatedAt = s.now().UTC()
	state.Sessions[index] = session
	state.Current = session.ID

	if err := s.save(state); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Store) load() (State, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	if len(data) == 0 {
		return State{}, nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse assist sessions: %w", err)
	}
	sortSessions(state.Sessions)
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

func normalizeSession(session Session) (Session, error) {
	session.Kind = Kind(strings.TrimSpace(string(session.Kind)))
	if session.Kind == "" {
		return Session{}, fmt.Errorf("assist session kind is required")
	}
	session.ScopeLabel = strings.TrimSpace(session.ScopeLabel)
	session.WorkspacePath = strings.TrimSpace(session.WorkspacePath)
	session.WorkareaName = strings.TrimSpace(session.WorkareaName)
	session.RepoTarget = strings.TrimSpace(session.RepoTarget)
	session.CommandTarget = strings.TrimSpace(session.CommandTarget)
	session.ConfigPath = strings.TrimSpace(session.ConfigPath)
	session.TicketID = strings.TrimSpace(session.TicketID)
	session.ReleaseID = strings.TrimSpace(session.ReleaseID)
	session.FromBranch = strings.TrimSpace(session.FromBranch)
	session.ToBranch = strings.TrimSpace(session.ToBranch)
	session.Audience = strings.TrimSpace(session.Audience)
	session.Mode = strings.TrimSpace(session.Mode)
	session.ThreadID = strings.TrimSpace(session.ThreadID)
	session.Summary = strings.TrimSpace(session.Summary)
	session.LastQuestion = strings.TrimSpace(session.LastQuestion)
	session.LastResponse = strings.TrimSpace(session.LastResponse)
	if session.ID == "" {
		session.ID = BuildID(session)
	}
	return session, nil
}

func currentSessionFromState(state State) (Session, bool, error) {
	if len(state.Sessions) == 0 {
		return Session{}, false, nil
	}
	if strings.TrimSpace(state.Current) == "" {
		return state.Sessions[0], true, nil
	}
	index := findIndex(state.Sessions, state.Current)
	if index < 0 {
		return state.Sessions[0], true, nil
	}
	return state.Sessions[index], true, nil
}

func findCurrentSessionForScope(sessions []Session, workareaName, repoTarget, workspacePath string) (Session, bool) {
	workareaName = strings.TrimSpace(workareaName)
	repoTarget = strings.TrimSpace(repoTarget)
	workspacePath = normalizeWorkspacePath(workspacePath)

	if workareaName != "" {
		for _, session := range sessions {
			if strings.EqualFold(strings.TrimSpace(session.WorkareaName), workareaName) {
				return session, true
			}
		}
	}
	if repoTarget != "" {
		for _, session := range sessions {
			if strings.TrimSpace(session.WorkareaName) != "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(session.RepoTarget), repoTarget) {
				return session, true
			}
		}
	}
	if workspacePath != "" {
		for _, session := range sessions {
			if strings.EqualFold(normalizeWorkspacePath(session.WorkspacePath), workspacePath) {
				return session, true
			}
		}
	}
	return Session{}, false
}

func findIndex(sessions []Session, id string) int {
	id = strings.TrimSpace(id)
	for index, session := range sessions {
		if strings.EqualFold(session.ID, id) {
			return index
		}
	}
	return -1
}

func sortSessions(sessions []Session) {
	sort.SliceStable(sessions, func(i, j int) bool {
		left := sessions[i].UpdatedAt
		right := sessions[j].UpdatedAt
		switch {
		case left.Equal(right):
			return strings.ToLower(sessions[i].ID) < strings.ToLower(sessions[j].ID)
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.After(right)
		}
	})
}
