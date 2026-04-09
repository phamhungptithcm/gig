package output

import (
	"fmt"
	"io"
	"strings"

	snapshotsvc "gig/internal/snapshot"
)

func RenderSnapshot(w io.Writer, snapshot snapshotsvc.TicketSnapshot, outputPath string) error {
	if _, err := fmt.Fprintf(w, "Ticket snapshot %s\n", snapshot.TicketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", snapshot.Workspace); err != nil {
		return err
	}
	if snapshot.ReleaseID != "" {
		if _, err := fmt.Fprintf(w, "Release: %s\n", snapshot.ReleaseID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Promotion baseline: %s -> %s\n", snapshot.FromBranch, snapshot.ToBranch); err != nil {
		return err
	}
	if len(snapshot.Environments) > 0 {
		if _, err := fmt.Fprintf(w, "Environments: %s\n", formatEnvironments(snapshot.Environments)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Captured at: %s\n", snapshot.CapturedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Tool version: %s\n", snapshot.ToolVersion); err != nil {
		return err
	}
	if snapshot.ConfigPath != "" {
		if _, err := fmt.Fprintf(w, "Config: %s\n", snapshot.ConfigPath); err != nil {
			return err
		}
	}
	if outputPath != "" {
		if _, err := fmt.Fprintf(w, "Saved to: %s\n", outputPath); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Plan verdict: %s\n", snapshot.Plan.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Verification verdict: %s\n", snapshot.Verification.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n", snapshot.Inspection.ScannedRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Touched repositories: %d\n", snapshot.Inspection.TouchedRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Recorded commits: %d\n\n", snapshot.Inspection.TotalCommits); err != nil {
		return err
	}

	if len(snapshot.Plan.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found for this ticket.")
		return err
	}

	inspectionByRoot := make(map[string]int, len(snapshot.Inspection.Repositories))
	for _, inspection := range snapshot.Inspection.Repositories {
		inspectionByRoot[inspection.Repository.Root] = len(inspection.Commits)
	}

	verificationByRoot := make(map[string]string, len(snapshot.Verification.Repositories))
	for _, repositoryVerification := range snapshot.Verification.Repositories {
		verificationByRoot[repositoryVerification.Repository.Root] = string(repositoryVerification.Verdict)
	}

	for i, repositoryPlan := range snapshot.Plan.Repositories {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", repositoryPlan.Repository.Name, repositoryPlan.Repository.Root, repositoryPlan.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  plan verdict: %s\n", repositoryPlan.Verdict); err != nil {
			return err
		}
		if verifyVerdict, ok := verificationByRoot[repositoryPlan.Repository.Root]; ok {
			if _, err := fmt.Fprintf(w, "  verify verdict: %s\n", verifyVerdict); err != nil {
				return err
			}
		}
		if commitCount, ok := inspectionByRoot[repositoryPlan.Repository.Root]; ok {
			if _, err := fmt.Fprintf(w, "  ticket commits recorded: %d\n", commitCount); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  commits to include: %d\n", len(repositoryPlan.Compare.MissingCommits)); err != nil {
			return err
		}
		if len(repositoryPlan.Branches) > 0 {
			if _, err := fmt.Fprintf(w, "  branches seen: %s\n", strings.Join(repositoryPlan.Branches, ", ")); err != nil {
				return err
			}
		}
		if len(repositoryPlan.EnvironmentStatuses) > 0 {
			if _, err := fmt.Fprintln(w, "  environment status:"); err != nil {
				return err
			}
			for _, status := range repositoryPlan.EnvironmentStatuses {
				if _, err := fmt.Fprintf(w, "    - %s\n", formatEnvironmentStatus(status)); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.DependencyResolutions) > 0 {
			if _, err := fmt.Fprintln(w, "  dependency status:"); err != nil {
				return err
			}
			for _, resolution := range repositoryPlan.DependencyResolutions {
				if _, err := fmt.Fprintf(w, "    - %s\n", formatDependencyResolution(resolution, snapshot.FromBranch, snapshot.ToBranch)); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Actions) > 0 {
			if _, err := fmt.Fprintln(w, "  planned actions:"); err != nil {
				return err
			}
			for _, action := range repositoryPlan.Actions {
				if _, err := fmt.Fprintf(w, "    - %s\n", action.Summary); err != nil {
					return err
				}
			}
		}

		if i < len(snapshot.Plan.Repositories)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}
