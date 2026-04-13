package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gig/internal/output"
)

type deerFlowDoctorDeps struct {
	lookPath    deerFlowSetupLookPath
	getenv      func(string) string
	healthCheck func(context.Context, string) error
}

func (a *App) runAssistDoctor(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printAssistDoctorUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	basePath := fs.String("path", ".", "Path to the gig repo root or deer-flow directory")
	deerflowURL := fs.String("url", "", "Base URL for DeerFlow, for example http://localhost:2026")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistDoctorUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "assist doctor does not accept positional arguments")
		a.printAssistDoctorUsage()
		return usageExitCode
	}

	selectedFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist doctor failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*basePath)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist doctor failed: %v\n", err)
		return 1
	}

	result, err := inspectDeerFlow(ctx, resolvedPath, *deerflowURL, deerFlowDoctorDeps{
		lookPath:    exec.LookPath,
		getenv:      os.Getenv,
		healthCheck: checkDeerFlowHealth,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist doctor failed: %v\n", err)
		return 1
	}

	switch selectedFormat {
	case outputFormatHuman:
		if err := output.RenderDeerFlowDoctor(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string                      `json:"command"`
			Result  output.DeerFlowDoctorResult `json:"result"`
		}{
			Command: "assist doctor",
			Result:  result,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func inspectDeerFlow(ctx context.Context, basePath, url string, deps deerFlowDoctorDeps) (output.DeerFlowDoctorResult, error) {
	root, err := findDeerFlowRoot(basePath)
	if err != nil {
		return output.DeerFlowDoctorResult{}, err
	}

	resolvedURL := strings.TrimRight(strings.TrimSpace(url), "/")
	if resolvedURL == "" {
		resolvedURL = "http://localhost:2026"
	}

	tools := detectDeerFlowTools(deps.lookPath)
	startCommand := buildDeerFlowStartCommand(root, tools)
	frontendEnvPath := ""
	if _, err := os.Stat(filepath.Join(root, "frontend", ".env")); err == nil {
		frontendEnvPath = filepath.Join(root, "frontend", ".env")
	}

	result := output.DeerFlowDoctorResult{
		Root:             root,
		URL:              resolvedURL,
		ConfigPath:       findExistingDeerFlowConfig(root),
		FrontendEnvPath:  frontendEnvPath,
		DockerAvailable:  tools.Docker,
		RecommendedStart: startCommand,
		Tools: []output.DeerFlowToolCheck{
			{Name: "make", Available: tools.Make},
			{Name: "docker", Available: tools.Docker},
			{Name: "uv", Available: tools.UV},
			{Name: "pnpm", Available: tools.PNPM},
			{Name: "python", Available: tools.Python || tools.Python3, Detail: pythonToolDetail(tools)},
		},
	}

	if deps.healthCheck != nil {
		if err := deps.healthCheck(ctx, resolvedURL); err == nil {
			result.GatewayHealthy = true
		} else {
			result.HealthError = err.Error()
		}
	}

	if result.ConfigPath == "" {
		result.Readiness = "setup-required"
		result.Next = append(result.Next, fmt.Sprintf("run gig assist setup --path %s", shellSingleQuote(root)))
		if !tools.Make {
			result.Next = append(result.Next, "install make so gig can start the local DeerFlow sidecar")
		}
		return result, nil
	}

	content, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		return output.DeerFlowDoctorResult{}, err
	}
	result.ModelConfigured = hasConfiguredDeerFlowModel(string(content))
	result.Credentials = collectDeerFlowCredentials(root, extractDeerFlowEnvVars(string(content)), deps.getenv)

	missingCredentials := 0
	for _, credential := range result.Credentials {
		if !credential.Present {
			missingCredentials++
			result.Next = append(result.Next, fmt.Sprintf("set %s before starting DeerFlow", credential.Name))
		}
	}

	if !result.ModelConfigured {
		result.Next = append(result.Next, fmt.Sprintf("add at least one model under models in %s", result.ConfigPath))
	}

	startableWithDocker := tools.Make && tools.Docker
	startableWithLocalDev := tools.Make && tools.UV && tools.PNPM && (tools.Python || tools.Python3)
	if !startableWithDocker && !startableWithLocalDev {
		result.Next = append(result.Next, "install docker, or install the local dev toolchain: make, python, uv, and pnpm")
	}

	if !result.GatewayHealthy && result.RecommendedStart != "" {
		result.Next = append(result.Next, fmt.Sprintf("start DeerFlow with %s", result.RecommendedStart))
	}

	switch {
	case !startableWithDocker && !startableWithLocalDev:
		result.Readiness = "tools-required"
	case !result.ModelConfigured || missingCredentials > 0:
		result.Readiness = "config-required"
	default:
		result.Readiness = "ready"
	}

	result.Next = uniqueSortedStrings(result.Next)
	return result, nil
}

func pythonToolDetail(tools deerFlowToolAvailability) string {
	switch {
	case tools.Python:
		return "python"
	case tools.Python3:
		return "python3"
	default:
		return ""
	}
}

func collectDeerFlowCredentials(root string, names []string, getenv func(string) string) []output.DeerFlowCredentialCheck {
	if len(names) == 0 {
		return nil
	}

	fileValues := map[string]string{}
	for _, envPath := range []string{
		filepath.Join(root, ".env"),
		filepath.Join(root, "frontend", ".env"),
	} {
		values, err := readDotEnvFile(envPath)
		if err != nil {
			continue
		}
		for key, value := range values {
			if _, exists := fileValues[key]; !exists {
				fileValues[key] = value
			}
		}
	}

	checks := make([]output.DeerFlowCredentialCheck, 0, len(names))
	for _, name := range names {
		check := output.DeerFlowCredentialCheck{Name: name}
		if getenv != nil && strings.TrimSpace(getenv(name)) != "" {
			check.Present = true
			check.Source = "shell"
		} else if value := strings.TrimSpace(fileValues[name]); value != "" {
			check.Present = true
			check.Source = ".env"
		}
		checks = append(checks, check)
	}

	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})
	return checks
}

func readDotEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "export ")
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(strings.Trim(value, `"'`))
	}
	return values, nil
}

func checkDeerFlowHealth(ctx context.Context, baseURL string) error {
	httpClient := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %s", resp.Status)
	}
	return nil
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	items := make([]string, 0, len(seen))
	for value := range seen {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}
