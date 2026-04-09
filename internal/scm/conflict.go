package scm

import "context"

type ConflictOperationType string

const (
	ConflictOperationMerge      ConflictOperationType = "merge"
	ConflictOperationRebase     ConflictOperationType = "rebase"
	ConflictOperationCherryPick ConflictOperationType = "cherry-pick"
)

type ConflictSide struct {
	Label      string   `json:"label"`
	Branch     string   `json:"branch,omitempty"`
	CommitHash string   `json:"commitHash,omitempty"`
	Subject    string   `json:"subject,omitempty"`
	TicketIDs  []string `json:"ticketIds,omitempty"`
}

func (s ConflictSide) ShortHash() string {
	if len(s.CommitHash) <= 8 {
		return s.CommitHash
	}

	return s.CommitHash[:8]
}

type ConflictOperationState struct {
	Type                ConflictOperationType `json:"type"`
	RepoRoot            string                `json:"repoRoot"`
	CurrentBranch       string                `json:"currentBranch,omitempty"`
	SequenceBranch      string                `json:"sequenceBranch,omitempty"`
	TargetBranch        string                `json:"targetBranch,omitempty"`
	ContinuationCommand string                `json:"continuationCommand,omitempty"`
	CurrentSide         ConflictSide          `json:"currentSide"`
	IncomingSide        ConflictSide          `json:"incomingSide"`
}

type ConflictFile struct {
	Path         string `json:"path"`
	ConflictCode string `json:"conflictCode"`
	BaseMode     string `json:"baseMode,omitempty"`
	CurrentMode  string `json:"currentMode,omitempty"`
	IncomingMode string `json:"incomingMode,omitempty"`
	BaseHash     string `json:"baseHash,omitempty"`
	CurrentHash  string `json:"currentHash,omitempty"`
	IncomingHash string `json:"incomingHash,omitempty"`
}

type ConflictProvider interface {
	ConflictState(ctx context.Context, repoRoot string) (ConflictOperationState, bool, error)
	ConflictFiles(ctx context.Context, repoRoot string) ([]ConflictFile, error)
	ConflictBlob(ctx context.Context, repoRoot, objectHash string) ([]byte, error)
	StageConflictFile(ctx context.Context, repoRoot, path string) error
}
