package toolcheck

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Kind string

const (
	KindMissingTool  Kind = "missing-tool"
	KindAuthRequired Kind = "auth-required"
)

type Tool struct {
	Name         string
	DisplayName  string
	Purpose      string
	LoginCommand string
	DocsURL      string
	Install      []InstallCommand
	EnvVars      []string
}

type InstallCommand struct {
	Platform string
	Command  string
}

type Status struct {
	Name         string           `json:"name"`
	DisplayName  string           `json:"displayName"`
	Required     bool             `json:"required"`
	Installed    bool             `json:"installed"`
	Path         string           `json:"path,omitempty"`
	Summary      string           `json:"summary"`
	LoginCommand string           `json:"loginCommand,omitempty"`
	DocsURL      string           `json:"docsUrl,omitempty"`
	Install      []InstallCommand `json:"install,omitempty"`
}

type DiagnosticError struct {
	Kind     Kind
	Tool     Tool
	Provider string
	Repo     string
	Message  string
	Cause    error
}

func MissingTool(tool Tool, cause error) error {
	tool = normalizeTool(tool)
	message := fmt.Sprintf("%s executable not found", tool.Name)
	return &DiagnosticError{
		Kind:    KindMissingTool,
		Tool:    tool,
		Message: message,
		Cause:   cause,
	}
}

func AuthRequired(provider, repo string, tool Tool, cause error) error {
	tool = normalizeTool(tool)
	message := strings.TrimSpace(provider) + " login is required"
	if strings.TrimSpace(repo) != "" {
		message += " for " + strings.TrimSpace(repo)
	}
	return &DiagnosticError{
		Kind:     KindAuthRequired,
		Tool:     tool,
		Provider: strings.TrimSpace(provider),
		Repo:     strings.TrimSpace(repo),
		Message:  message,
		Cause:    cause,
	}
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}

	lines := []string{strings.TrimSpace(e.Message)}
	if e.Kind == KindMissingTool {
		if e.Cause != nil {
			lines[0] = fmt.Sprintf("%s: %v", lines[0], e.Cause)
		}
		if e.Tool.Purpose != "" {
			lines = append(lines, e.Tool.Purpose)
		}
		if len(e.Tool.Install) > 0 {
			lines = append(lines, "Install:")
			for _, install := range e.Tool.Install {
				if strings.TrimSpace(install.Platform) == "" || strings.TrimSpace(install.Command) == "" {
					continue
				}
				lines = append(lines, fmt.Sprintf("  %s: %s", install.Platform, install.Command))
			}
		}
	}

	next := e.nextCommands()
	if len(next) > 0 {
		lines = append(lines, "Next:")
		for _, command := range next {
			lines = append(lines, "  "+command)
		}
	}
	if len(e.Tool.EnvVars) > 0 {
		lines = append(lines, "Environment:")
		for _, name := range e.Tool.EnvVars {
			lines = append(lines, "  "+name)
		}
	}
	if e.Tool.DocsURL != "" {
		lines = append(lines, "Docs: "+e.Tool.DocsURL)
	}
	if e.Kind == KindAuthRequired && e.Cause != nil {
		lines = append(lines, "Details: "+e.Cause.Error())
	}

	return strings.Join(lines, "\n")
}

func (e *DiagnosticError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *DiagnosticError) nextCommands() []string {
	if e == nil {
		return nil
	}
	command := strings.TrimSpace(e.Tool.LoginCommand)
	if command == "" {
		return nil
	}
	return []string{command}
}

func IsMissingTool(err error) bool {
	var diagnostic *DiagnosticError
	return errors.As(err, &diagnostic) && diagnostic.Kind == KindMissingTool
}

func IsAuthRequired(err error) bool {
	var diagnostic *DiagnosticError
	return errors.As(err, &diagnostic) && diagnostic.Kind == KindAuthRequired
}

func Detail(err error) string {
	if err == nil {
		return "ready"
	}
	var diagnostic *DiagnosticError
	if errors.As(err, &diagnostic) {
		switch diagnostic.Kind {
		case KindMissingTool:
			return "cli missing"
		case KindAuthRequired:
			return "login required"
		}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "executable not found"):
		return "cli missing"
	case strings.Contains(message, "credential"), strings.Contains(message, "token"), strings.Contains(message, "password"):
		return "credentials needed"
	default:
		return "login required"
	}
}

func Git() Tool {
	return Tool{
		Name:        "git",
		DisplayName: "Git",
		Purpose:     "Git is required for local workspace inspection.",
		DocsURL:     "https://git-scm.com/downloads",
		Install: []InstallCommand{
			{Platform: "macOS", Command: "brew install git"},
			{Platform: "Windows winget", Command: "winget install --id Git.Git"},
			{Platform: "Windows Scoop", Command: "scoop install git"},
			{Platform: "Windows Chocolatey", Command: "choco install git"},
			{Platform: "Debian/Ubuntu", Command: "sudo apt install git"},
			{Platform: "Fedora", Command: "sudo dnf install git"},
			{Platform: "Arch", Command: "sudo pacman -S git"},
		},
	}
}

func GitHubCLI() Tool {
	return Tool{
		Name:         "gh",
		DisplayName:  "GitHub CLI",
		Purpose:      "GitHub CLI is required for GitHub repository access.",
		LoginCommand: "gig login github",
		DocsURL:      "https://cli.github.com/",
		Install: []InstallCommand{
			{Platform: "macOS", Command: "brew install gh"},
			{Platform: "Windows winget", Command: "winget install --id GitHub.cli"},
			{Platform: "Windows Scoop", Command: "scoop install gh"},
			{Platform: "Windows Chocolatey", Command: "choco install gh"},
			{Platform: "Debian/Ubuntu", Command: "sudo apt install gh"},
			{Platform: "Fedora", Command: "sudo dnf install gh"},
			{Platform: "Arch", Command: "sudo pacman -S github-cli"},
		},
	}
}

func GitLabCLI() Tool {
	return Tool{
		Name:         "glab",
		DisplayName:  "GitLab CLI",
		Purpose:      "GitLab CLI is required for GitLab repository access.",
		LoginCommand: "gig login gitlab",
		DocsURL:      "https://gitlab.com/gitlab-org/cli",
		Install: []InstallCommand{
			{Platform: "macOS", Command: "brew install glab"},
			{Platform: "Windows winget", Command: "winget install --id GitLab.cli"},
			{Platform: "Windows Scoop", Command: "scoop install glab"},
			{Platform: "Windows Chocolatey", Command: "choco install glab"},
			{Platform: "Debian/Ubuntu", Command: "sudo apt install glab"},
			{Platform: "Fedora", Command: "sudo dnf install glab"},
			{Platform: "Arch", Command: "sudo pacman -S glab"},
		},
	}
}

func AzureCLI() Tool {
	return Tool{
		Name:         "az",
		DisplayName:  "Azure CLI",
		Purpose:      "Azure CLI is required for Azure DevOps repository access.",
		LoginCommand: "gig login azure-devops",
		DocsURL:      "https://learn.microsoft.com/cli/azure/install-azure-cli",
		Install: []InstallCommand{
			{Platform: "macOS", Command: "brew install azure-cli"},
			{Platform: "Windows winget", Command: "winget install --id Microsoft.AzureCLI"},
			{Platform: "Windows Scoop", Command: "scoop install azure-cli"},
			{Platform: "Windows Chocolatey", Command: "choco install azure-cli"},
			{Platform: "Debian/Ubuntu", Command: "sudo apt install azure-cli"},
			{Platform: "Fedora", Command: "sudo dnf install azure-cli"},
			{Platform: "Arch", Command: "sudo pacman -S azure-cli"},
		},
	}
}

func SVN() Tool {
	return Tool{
		Name:         "svn",
		DisplayName:  "Subversion",
		Purpose:      "Subversion is required for SVN repository access.",
		LoginCommand: "gig login svn",
		DocsURL:      "https://subversion.apache.org/packages.html",
		EnvVars:      []string{"GIG_SVN_USERNAME", "GIG_SVN_PASSWORD"},
		Install: []InstallCommand{
			{Platform: "macOS", Command: "brew install subversion"},
			{Platform: "Windows winget", Command: "winget install --id Apache.Subversion"},
			{Platform: "Windows Scoop", Command: "scoop install svn"},
			{Platform: "Windows Chocolatey", Command: "choco install svn"},
			{Platform: "Debian/Ubuntu", Command: "sudo apt install subversion"},
			{Platform: "Fedora", Command: "sudo dnf install subversion"},
			{Platform: "Arch", Command: "sudo pacman -S subversion"},
		},
	}
}

func Bitbucket() Tool {
	return Tool{
		Name:         "bitbucket-api-token",
		DisplayName:  "Bitbucket API token",
		Purpose:      "Bitbucket API credentials are required for Bitbucket repository access.",
		LoginCommand: "gig login bitbucket",
		DocsURL:      "https://support.atlassian.com/bitbucket-cloud/docs/create-an-api-token/",
		EnvVars:      []string{"GIG_BITBUCKET_EMAIL", "GIG_BITBUCKET_API_TOKEN"},
	}
}

func SystemTools() []Tool {
	return []Tool{
		Git(),
		GitHubCLI(),
		GitLabCLI(),
		AzureCLI(),
		SVN(),
	}
}

func CheckSystemDependencies(required map[string]bool) []Status {
	return CheckSystemDependenciesWithLookPath(exec.LookPath, required)
}

func CheckSystemDependenciesWithLookPath(lookPath func(string) (string, error), required map[string]bool) []Status {
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	tools := SystemTools()
	checks := make([]Status, 0, len(tools))
	for _, tool := range tools {
		tool = normalizeTool(tool)
		path, err := lookPath(tool.Name)
		installed := err == nil
		status := Status{
			Name:         tool.Name,
			DisplayName:  tool.DisplayName,
			Required:     required[tool.Name],
			Installed:    installed,
			Path:         strings.TrimSpace(path),
			LoginCommand: tool.LoginCommand,
			DocsURL:      tool.DocsURL,
			Install:      tool.Install,
		}
		switch {
		case installed && status.Required:
			status.Summary = tool.DisplayName + " is installed."
		case installed:
			status.Summary = tool.DisplayName + " is installed for optional provider workflows."
		case status.Required:
			status.Summary = tool.DisplayName + " is missing and required for the selected workflow."
		default:
			status.Summary = tool.DisplayName + " is missing; install it only when using that provider."
		}
		checks = append(checks, status)
	}
	return checks
}

func MissingRequired(checks []Status) []Status {
	missing := make([]Status, 0)
	for _, check := range checks {
		if check.Required && !check.Installed {
			missing = append(missing, check)
		}
	}
	return missing
}

func MissingOptional(checks []Status) []Status {
	missing := make([]Status, 0)
	for _, check := range checks {
		if !check.Required && !check.Installed {
			missing = append(missing, check)
		}
	}
	return missing
}

func PreferredInstallCommand(status Status) (InstallCommand, bool) {
	return preferredInstallCommand(runtime.GOOS, status.Install)
}

func preferredInstallCommand(goos string, installs []InstallCommand) (InstallCommand, bool) {
	preferred := []string{}
	switch goos {
	case "darwin":
		preferred = []string{"macOS"}
	case "windows":
		preferred = []string{"Windows winget", "Windows Scoop", "Windows Chocolatey"}
	case "linux":
		preferred = []string{"Debian/Ubuntu", "Fedora", "Arch"}
	}
	for _, platform := range preferred {
		for _, install := range installs {
			if strings.EqualFold(install.Platform, platform) && strings.TrimSpace(install.Command) != "" {
				return install, true
			}
		}
	}
	for _, install := range installs {
		if strings.TrimSpace(install.Command) != "" {
			return install, true
		}
	}
	return InstallCommand{}, false
}

func normalizeTool(tool Tool) Tool {
	tool.Name = strings.TrimSpace(tool.Name)
	tool.DisplayName = strings.TrimSpace(tool.DisplayName)
	tool.Purpose = strings.TrimSpace(tool.Purpose)
	tool.LoginCommand = strings.TrimSpace(tool.LoginCommand)
	tool.DocsURL = strings.TrimSpace(tool.DocsURL)
	if tool.DisplayName == "" {
		tool.DisplayName = tool.Name
	}
	return tool
}
