package conflict

import "gig/internal/scm"

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Status struct {
	Repository       scm.Repository             `json:"repository"`
	Operation        scm.ConflictOperationState `json:"operation"`
	Files            []FileStatus               `json:"files,omitempty"`
	ResolvableFiles  int                        `json:"resolvableFiles"`
	UnsupportedFiles int                        `json:"unsupportedFiles"`
	ScopeTicketID    string                     `json:"scopeTicketId,omitempty"`
	SuggestedNext    string                     `json:"suggestedNext,omitempty"`
}

type FileStatus struct {
	Path              string   `json:"path"`
	ConflictCode      string   `json:"conflictCode"`
	BlockCount        int      `json:"blockCount"`
	Supported         bool     `json:"supported"`
	UnsupportedReason string   `json:"unsupportedReason,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}

type Session struct {
	Status       Status
	CurrentFile  string
	CurrentBlock int
}

type ParsedFile struct {
	Segments []Segment
	Blocks   []Block
}

type Segment struct {
	Text  []byte
	Block *Block
}

type Block struct {
	Index       int
	StartLine   int
	EndLine     int
	Current     []byte
	Base        []byte
	Incoming    []byte
	CurrentRef  scm.ConflictSide
	IncomingRef scm.ConflictSide
}

type RiskAssessment struct {
	Severity       Severity `json:"severity"`
	Summary        string   `json:"summary"`
	ReviewNotes    []string `json:"reviewNotes,omitempty"`
	DuplicateLines []string `json:"duplicateLines,omitempty"`
}

type ActiveConflict struct {
	Repository    scm.Repository             `json:"repository"`
	Operation     scm.ConflictOperationState `json:"operation"`
	File          FileStatus                 `json:"file"`
	Block         Block                      `json:"block"`
	Risk          RiskAssessment             `json:"risk"`
	ScopeTicketID string                     `json:"scopeTicketId,omitempty"`
	ScopeWarnings []string                   `json:"scopeWarnings,omitempty"`
}

type ResolutionChoice string

const (
	ResolutionCurrent           ResolutionChoice = "current"
	ResolutionIncoming          ResolutionChoice = "incoming"
	ResolutionBothCurrentFirst  ResolutionChoice = "both-current-first"
	ResolutionBothIncomingFirst ResolutionChoice = "both-incoming-first"
)
