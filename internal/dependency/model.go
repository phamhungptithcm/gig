package dependency

const TrailerDependsOn = "Depends-On"

type DeclaredDependency struct {
	TicketID      string
	DependsOn     string
	CommitHash    string
	CommitSubject string
	TrailerKey    string
}

type Status string

const (
	StatusSatisfied     Status = "satisfied"
	StatusMissingTarget Status = "missing-target"
	StatusUnresolved    Status = "unresolved"
)

type Resolution struct {
	TicketID        string
	DependsOn       string
	Status          Status
	FoundInSource   bool
	FoundInTarget   bool
	EvidenceCommits []string
}
