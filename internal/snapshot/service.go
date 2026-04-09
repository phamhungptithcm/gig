package snapshot

import (
	"context"
	"strings"
	"time"

	"gig/internal/buildinfo"
	"gig/internal/config"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
)

const SchemaVersion = "1"

type inspector interface {
	Inspect(ctx context.Context, path, ticketID string) ([]inspectsvc.RepositoryInspection, int, error)
}

type planner interface {
	BuildPromotionPlan(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (plansvc.PromotionPlan, error)
	VerifyPromotion(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (plansvc.Verification, error)
}

type InspectionSnapshot struct {
	ScannedRepositories int                               `json:"scannedRepositories"`
	TouchedRepositories int                               `json:"touchedRepositories"`
	TotalCommits        int                               `json:"totalCommits"`
	Repositories        []inspectsvc.RepositoryInspection `json:"repositories,omitempty"`
}

type TicketSnapshot struct {
	SchemaVersion string                   `json:"schemaVersion"`
	ReleaseID     string                   `json:"releaseId,omitempty"`
	CapturedAt    time.Time                `json:"capturedAt"`
	ToolVersion   string                   `json:"toolVersion"`
	Workspace     string                   `json:"workspace"`
	ConfigPath    string                   `json:"configPath,omitempty"`
	TicketID      string                   `json:"ticketId"`
	FromBranch    string                   `json:"fromBranch"`
	ToBranch      string                   `json:"toBranch"`
	Environments  []inspectsvc.Environment `json:"environments,omitempty"`
	Inspection    InspectionSnapshot       `json:"inspection"`
	Plan          plansvc.PromotionPlan    `json:"plan"`
	Verification  plansvc.Verification     `json:"verification"`
}

type CaptureOptions struct {
	ReleaseID string
}

type Service struct {
	inspector inspector
	planner   planner
	now       func() time.Time
}

func NewService(inspector inspector, planner planner) *Service {
	return &Service{
		inspector: inspector,
		planner:   planner,
		now:       time.Now,
	}
}

func (s *Service) Capture(ctx context.Context, workspacePath string, loaded config.Loaded, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (TicketSnapshot, error) {
	return s.CaptureWithOptions(ctx, workspacePath, loaded, ticketID, fromBranch, toBranch, environments, CaptureOptions{})
}

func (s *Service) CaptureWithOptions(ctx context.Context, workspacePath string, loaded config.Loaded, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment, options CaptureOptions) (TicketSnapshot, error) {
	inspections, scannedRepositories, err := s.inspector.Inspect(ctx, workspacePath, ticketID)
	if err != nil {
		return TicketSnapshot{}, err
	}

	promotionPlan, err := s.planner.BuildPromotionPlan(ctx, workspacePath, ticketID, fromBranch, toBranch, environments)
	if err != nil {
		return TicketSnapshot{}, err
	}

	verification, err := s.planner.VerifyPromotion(ctx, workspacePath, ticketID, fromBranch, toBranch, environments)
	if err != nil {
		return TicketSnapshot{}, err
	}

	totalCommits := 0
	for _, inspection := range inspections {
		totalCommits += len(inspection.Commits)
	}

	capturedAt := time.Now().UTC()
	if s.now != nil {
		capturedAt = s.now().UTC()
	}

	return TicketSnapshot{
		SchemaVersion: SchemaVersion,
		ReleaseID:     strings.TrimSpace(options.ReleaseID),
		CapturedAt:    capturedAt,
		ToolVersion:   buildinfo.Summary(),
		Workspace:     workspacePath,
		ConfigPath:    loaded.Path,
		TicketID:      ticketID,
		FromBranch:    fromBranch,
		ToBranch:      toBranch,
		Environments:  cloneEnvironments(environments),
		Inspection: InspectionSnapshot{
			ScannedRepositories: scannedRepositories,
			TouchedRepositories: len(inspections),
			TotalCommits:        totalCommits,
			Repositories:        cloneInspections(inspections),
		},
		Plan:         promotionPlan,
		Verification: verification,
	}, nil
}

func cloneEnvironments(environments []inspectsvc.Environment) []inspectsvc.Environment {
	if len(environments) == 0 {
		return nil
	}

	cloned := make([]inspectsvc.Environment, len(environments))
	copy(cloned, environments)
	return cloned
}

func cloneInspections(inspections []inspectsvc.RepositoryInspection) []inspectsvc.RepositoryInspection {
	if len(inspections) == 0 {
		return nil
	}

	cloned := make([]inspectsvc.RepositoryInspection, len(inspections))
	copy(cloned, inspections)
	return cloned
}
