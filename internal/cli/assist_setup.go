package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gig/internal/output"
)

type deerFlowSetupLookPath func(file string) (string, error)

type deerFlowSetupDeps struct {
	lookPath deerFlowSetupLookPath
}

func (a *App) runAssistSetup(ctx context.Context, args []string) int {
	_ = ctx
	if hasHelpFlag(args) {
		a.printAssistSetupUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	basePath := fs.String("path", ".", "Path to the gig repo root or deer-flow directory")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistSetupUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "assist setup does not accept positional arguments")
		a.printAssistSetupUsage()
		return usageExitCode
	}

	selectedFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist setup failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*basePath)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist setup failed: %v\n", err)
		return 1
	}

	result, err := bootstrapDeerFlow(resolvedPath, deerFlowSetupDeps{
		lookPath: exec.LookPath,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist setup failed: %v\n", err)
		return 1
	}

	switch selectedFormat {
	case outputFormatHuman:
		if err := output.RenderDeerFlowSetup(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string                     `json:"command"`
			Result  output.DeerFlowSetupResult `json:"result"`
		}{
			Command: "assist setup",
			Result:  result,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func bootstrapDeerFlow(basePath string, deps deerFlowSetupDeps) (output.DeerFlowSetupResult, error) {
	root, err := findDeerFlowRoot(basePath)
	if err != nil {
		return output.DeerFlowSetupResult{}, err
	}

	configPath, createdFiles, err := ensureDeerFlowConfig(root)
	if err != nil {
		return output.DeerFlowSetupResult{}, err
	}

	tools := detectDeerFlowTools(deps.lookPath)
	startCommand := buildDeerFlowStartCommand(root, tools)

	remaining, err := deerFlowRemainingActions(configPath)
	if err != nil {
		return output.DeerFlowSetupResult{}, err
	}

	return output.DeerFlowSetupResult{
		Root:             root,
		ConfigPath:       configPath,
		CreatedFiles:     createdFiles,
		DockerAvailable:  tools.Docker,
		RecommendedStart: startCommand,
		Remaining:        remaining,
	}, nil
}

func findDeerFlowRoot(basePath string) (string, error) {
	candidates := []string{
		strings.TrimSpace(basePath),
		filepath.Join(strings.TrimSpace(basePath), "deer-flow"),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if looksLikeDeerFlowRoot(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not find a deer-flow directory under %s", basePath)
}

func looksLikeDeerFlowRoot(root string) bool {
	required := []string{
		"Makefile",
		"backend",
		"frontend",
		"config.example.yaml",
	}

	for _, name := range required {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return false
		}
	}
	return true
}

func findExistingDeerFlowConfig(root string) string {
	candidates := []string{
		filepath.Join(root, "config.yaml"),
		filepath.Join(root, "config.yml"),
		filepath.Join(root, "configure.yml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func ensureDeerFlowConfig(root string) (string, []string, error) {
	if existing := findExistingDeerFlowConfig(root); existing != "" {
		return existing, nil, nil
	}

	created := make([]string, 0, 3)
	copies := [][2]string{
		{filepath.Join(root, "config.example.yaml"), filepath.Join(root, "config.yaml")},
		{filepath.Join(root, ".env.example"), filepath.Join(root, ".env")},
		{filepath.Join(root, "frontend", ".env.example"), filepath.Join(root, "frontend", ".env")},
	}

	for _, pair := range copies {
		src := pair[0]
		dst := pair[1]
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, err
		}
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		content, err := os.ReadFile(src)
		if err != nil {
			return "", nil, err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return "", nil, err
		}
		created = append(created, dst)
	}

	return filepath.Join(root, "config.yaml"), created, nil
}

type deerFlowToolAvailability struct {
	Make    bool
	Docker  bool
	UV      bool
	PNPM    bool
	Python  bool
	Python3 bool
}

func detectDeerFlowTools(lookPath deerFlowSetupLookPath) deerFlowToolAvailability {
	return deerFlowToolAvailability{
		Make:    lookupAvailable(lookPath, "make"),
		Docker:  lookupAvailable(lookPath, "docker"),
		UV:      lookupAvailable(lookPath, "uv"),
		PNPM:    lookupAvailable(lookPath, "pnpm"),
		Python:  lookupAvailable(lookPath, "python"),
		Python3: lookupAvailable(lookPath, "python3"),
	}
}

func buildDeerFlowStartCommand(root string, tools deerFlowToolAvailability) string {
	if !tools.Make {
		return ""
	}
	if tools.Docker {
		return fmt.Sprintf("cd %s && make docker-start", shellSingleQuote(root))
	}
	if tools.UV && tools.PNPM {
		switch {
		case tools.Python:
			return fmt.Sprintf("cd %s && make dev", shellSingleQuote(root))
		case tools.Python3:
			return fmt.Sprintf("cd %s && PYTHON=python3 make dev", shellSingleQuote(root))
		}
	}
	return ""
}

func deerFlowRemainingActions(configPath string) ([]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	remaining := make([]string, 0, 4)
	if !hasConfiguredDeerFlowModel(string(content)) {
		remaining = append(remaining, fmt.Sprintf("add at least one model under models in %s", configPath))
	}

	varNames := extractDeerFlowEnvVars(string(content))
	for _, name := range varNames {
		remaining = append(remaining, fmt.Sprintf("set %s with a real model credential before starting DeerFlow", name))
	}

	return remaining, nil
}

func hasConfiguredDeerFlowModel(content string) bool {
	lines := strings.Split(content, "\n")
	inModels := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inModels {
			if trimmed == "models:" {
				inModels = true
			}
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			return false
		}
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "- name:") {
			return true
		}
	}
	return false
}

func extractDeerFlowEnvVars(content string) []string {
	lines := strings.Split(content, "\n")
	pattern := regexp.MustCompile(`\$[A-Z][A-Z0-9_]+`)
	seen := map[string]struct{}{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, match := range pattern.FindAllString(line, -1) {
			seen[strings.TrimPrefix(match, "$")] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func lookupAvailable(lookPath deerFlowSetupLookPath, name string) bool {
	if lookPath == nil {
		return false
	}
	_, err := lookPath(name)
	return err == nil
}
