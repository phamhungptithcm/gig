package output

import (
	"fmt"
	"io"
)

type DeerFlowToolCheck struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Detail    string `json:"detail,omitempty"`
}

type DeerFlowCredentialCheck struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Source  string `json:"source,omitempty"`
}

type DeerFlowDoctorResult struct {
	Root             string                    `json:"root"`
	URL              string                    `json:"url"`
	Readiness        string                    `json:"readiness"`
	ConfigPath       string                    `json:"configPath,omitempty"`
	FrontendEnvPath  string                    `json:"frontendEnvPath,omitempty"`
	ModelConfigured  bool                      `json:"modelConfigured"`
	DockerAvailable  bool                      `json:"dockerAvailable"`
	GatewayHealthy   bool                      `json:"gatewayHealthy"`
	HealthError      string                    `json:"healthError,omitempty"`
	Tools            []DeerFlowToolCheck       `json:"tools,omitempty"`
	Credentials      []DeerFlowCredentialCheck `json:"credentials,omitempty"`
	RecommendedStart string                    `json:"recommendedStart,omitempty"`
	Next             []string                  `json:"next,omitempty"`
}

func RenderDeerFlowDoctor(w io.Writer, result DeerFlowDoctorResult) error {
	if _, err := fmt.Fprintln(w, "DeerFlow doctor"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Root: %s\n", result.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Readiness: %s\n", result.Readiness); err != nil {
		return err
	}
	if result.ConfigPath != "" {
		if _, err := fmt.Fprintf(w, "Config: %s\n", result.ConfigPath); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "Config: missing"); err != nil {
			return err
		}
	}
	if result.FrontendEnvPath != "" {
		if _, err := fmt.Fprintf(w, "Frontend env: %s\n", result.FrontendEnvPath); err != nil {
			return err
		}
	}

	gatewayStatus := "offline"
	if result.GatewayHealthy {
		gatewayStatus = "reachable"
	}
	if _, err := fmt.Fprintf(w, "Gateway: %s (%s)\n", gatewayStatus, result.URL); err != nil {
		return err
	}
	if result.HealthError != "" {
		if _, err := fmt.Fprintf(w, "Health: %s\n", result.HealthError); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Model configured: %s\n", yesNo(result.ModelConfigured)); err != nil {
		return err
	}
	if result.RecommendedStart != "" {
		if _, err := fmt.Fprintf(w, "Start: %s\n", result.RecommendedStart); err != nil {
			return err
		}
	}

	if len(result.Tools) > 0 {
		if _, err := fmt.Fprintln(w, "Tools:"); err != nil {
			return err
		}
		for _, tool := range result.Tools {
			status := "missing"
			if tool.Available {
				status = "found"
			}
			if tool.Detail != "" {
				if _, err := fmt.Fprintf(w, "  - %s: %s (%s)\n", tool.Name, status, tool.Detail); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "  - %s: %s\n", tool.Name, status); err != nil {
				return err
			}
		}
	}

	if len(result.Credentials) > 0 {
		if _, err := fmt.Fprintln(w, "Credentials:"); err != nil {
			return err
		}
		for _, credential := range result.Credentials {
			status := "missing"
			if credential.Present {
				status = "present"
			}
			if credential.Source != "" {
				if _, err := fmt.Fprintf(w, "  - %s: %s (%s)\n", credential.Name, status, credential.Source); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "  - %s: %s\n", credential.Name, status); err != nil {
				return err
			}
		}
	}

	if len(result.Next) > 0 {
		if _, err := fmt.Fprintln(w, "Next:"); err != nil {
			return err
		}
		for _, item := range result.Next {
			if _, err := fmt.Fprintf(w, "  - %s\n", item); err != nil {
				return err
			}
		}
	}

	return nil
}
