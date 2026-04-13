package output

import (
	"fmt"
	"io"
)

type DeerFlowSetupResult struct {
	Root             string   `json:"root"`
	ConfigPath       string   `json:"configPath"`
	CreatedFiles     []string `json:"createdFiles,omitempty"`
	DockerAvailable  bool     `json:"dockerAvailable"`
	RecommendedStart string   `json:"recommendedStart"`
	Remaining        []string `json:"remaining,omitempty"`
}

func RenderDeerFlowSetup(w io.Writer, result DeerFlowSetupResult) error {
	if _, err := fmt.Fprintln(w, "DeerFlow setup"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Root: %s\n", result.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Config: %s\n", result.ConfigPath); err != nil {
		return err
	}
	if len(result.CreatedFiles) > 0 {
		if _, err := fmt.Fprintln(w, "Created:"); err != nil {
			return err
		}
		for _, path := range result.CreatedFiles {
			if _, err := fmt.Fprintf(w, "  - %s\n", path); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(w, "Docker: %s\n", yesNo(result.DockerAvailable)); err != nil {
		return err
	}
	if result.RecommendedStart != "" {
		if _, err := fmt.Fprintf(w, "Next: %s\n", result.RecommendedStart); err != nil {
			return err
		}
	}
	if len(result.Remaining) > 0 {
		if _, err := fmt.Fprintln(w, "Remaining:"); err != nil {
			return err
		}
		for _, item := range result.Remaining {
			if _, err := fmt.Fprintf(w, "  - %s\n", item); err != nil {
				return err
			}
		}
	}
	return nil
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
