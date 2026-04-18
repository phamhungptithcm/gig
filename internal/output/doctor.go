package output

import (
	"fmt"
	"io"
	"strings"

	doctorsvc "gig/internal/doctor"
)

func RenderDoctor(w io.Writer, report doctorsvc.Report) error {
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", report.Workspace); err != nil {
		return err
	}
	if report.ConfigPath != "" {
		if _, err := fmt.Fprintf(w, "Overrides: %s\n", report.ConfigPath); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "Overrides: none (using built-in inference and defaults)"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Verdict: %s\n", report.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n", report.Summary.ScannedRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Configured repositories: %d\n", report.Summary.ConfiguredRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Covered repositories: %d\n", report.Summary.CoveredRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Missing catalog entries: %d\n", report.Summary.MissingCatalogEntries); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Missing environment refs: %d\n\n", report.Summary.MissingEnvironmentRefs); err != nil {
		return err
	}

	if len(report.Findings) > 0 {
		if _, err := fmt.Fprintln(w, "Workspace Findings"); err != nil {
			return err
		}
		for _, finding := range report.Findings {
			if _, err := fmt.Fprintf(w, "  - [%s] %s\n", finding.Severity, finding.Summary); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(report.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repositories found under the selected workspace.")
		return err
	}

	for i, repository := range report.Repositories {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", repository.Repository.Name, repository.Repository.Root, repository.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  verdict: %s\n", repository.Verdict); err != nil {
			return err
		}
		if repository.ConfigEntry != nil {
			if repository.ConfigEntry.Service != "" {
				if _, err := fmt.Fprintf(w, "  service: %s\n", repository.ConfigEntry.Service); err != nil {
					return err
				}
			}
			if repository.ConfigEntry.Owner != "" {
				if _, err := fmt.Fprintf(w, "  owner: %s\n", repository.ConfigEntry.Owner); err != nil {
					return err
				}
			}
			if repository.ConfigEntry.Kind != "" {
				if _, err := fmt.Fprintf(w, "  kind: %s\n", repository.ConfigEntry.Kind); err != nil {
					return err
				}
			}
		}
		if repository.Capabilities != nil {
			if _, err := fmt.Fprintf(w, "  provider capabilities: %s\n", repository.Capabilities.Summary()); err != nil {
				return err
			}
		}
		if repository.Topology != nil {
			if _, err := fmt.Fprintf(w, "  topology confidence: %s\n", repository.Topology.Confidence); err != nil {
				return err
			}
			if repository.Topology.Summary != "" {
				if _, err := fmt.Fprintf(w, "  topology summary: %s\n", repository.Topology.Summary); err != nil {
					return err
				}
			}
			if len(repository.Topology.ProtectedBranches) > 0 {
				if _, err := fmt.Fprintf(w, "  protected branches: %s\n", strings.Join(repository.Topology.ProtectedBranches, ", ")); err != nil {
					return err
				}
			}
		}
		if len(repository.EnvironmentChecks) > 0 {
			if _, err := fmt.Fprintln(w, "  environment checks:"); err != nil {
				return err
			}
			for _, check := range repository.EnvironmentChecks {
				status := "missing"
				if check.Exists {
					status = "found"
				}
				if _, err := fmt.Fprintf(w, "    - %s=%s: %s\n", check.Environment.Name, check.Environment.Branch, status); err != nil {
					return err
				}
			}
		}
		if len(repository.Findings) > 0 {
			if _, err := fmt.Fprintln(w, "  findings:"); err != nil {
				return err
			}
			for _, finding := range repository.Findings {
				if _, err := fmt.Fprintf(w, "    - [%s] %s\n", finding.Severity, finding.Summary); err != nil {
					return err
				}
			}
		}

		if i < len(report.Repositories)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}
