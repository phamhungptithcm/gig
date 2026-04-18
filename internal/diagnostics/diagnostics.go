package diagnostics

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const envDiagnosticsFile = "GIG_DIAGNOSTICS_FILE"

type Logger struct {
	filePath string
	mu       sync.Mutex
}

type Meta struct {
	Command       string         `json:"command,omitempty"`
	WorkspacePath string         `json:"workspacePath,omitempty"`
	Repo          string         `json:"repo,omitempty"`
	SCM           string         `json:"scm,omitempty"`
	Ticket        string         `json:"ticket,omitempty"`
	FromBranch    string         `json:"fromBranch,omitempty"`
	ToBranch      string         `json:"toBranch,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
}

type Event struct {
	Timestamp  string         `json:"timestamp"`
	Level      string         `json:"level"`
	Operation  string         `json:"operation"`
	Message    string         `json:"message"`
	Error      string         `json:"error,omitempty"`
	Command    string         `json:"command,omitempty"`
	Workspace  string         `json:"workspace,omitempty"`
	Repo       string         `json:"repo,omitempty"`
	SCM        string         `json:"scm,omitempty"`
	Ticket     string         `json:"ticket,omitempty"`
	FromBranch string         `json:"fromBranch,omitempty"`
	ToBranch   string         `json:"toBranch,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

type loggerContextKey struct{}

func NewFromEnv(lookup func(string) (string, bool)) *Logger {
	if lookup == nil {
		return nil
	}
	value, ok := lookup(envDiagnosticsFile)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &Logger{filePath: value}
}

func WithLogger(ctx context.Context, logger *Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func Emit(ctx context.Context, level, operation, message string, meta Meta, err error) {
	logger, _ := ctx.Value(loggerContextKey{}).(*Logger)
	if logger == nil {
		return
	}

	event := Event{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Level:      strings.TrimSpace(level),
		Operation:  strings.TrimSpace(operation),
		Message:    strings.TrimSpace(message),
		Command:    strings.TrimSpace(meta.Command),
		Workspace:  strings.TrimSpace(meta.WorkspacePath),
		Repo:       strings.TrimSpace(meta.Repo),
		SCM:        strings.TrimSpace(meta.SCM),
		Ticket:     strings.TrimSpace(meta.Ticket),
		FromBranch: strings.TrimSpace(meta.FromBranch),
		ToBranch:   strings.TrimSpace(meta.ToBranch),
		Details:    meta.Details,
	}
	if err != nil {
		event.Error = err.Error()
	}
	logger.write(event)
}

func (l *Logger) write(event Event) {
	if l == nil || strings.TrimSpace(l.filePath) == "" {
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	payload = append(payload, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.filePath), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(payload)
}
