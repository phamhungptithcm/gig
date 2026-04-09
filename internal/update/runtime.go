package update

import (
	"path/filepath"
	"strings"
)

type InstallMode string

const (
	ModeDirect   InstallMode = "direct"
	ModeHomebrew InstallMode = "homebrew"
	ModeScoop    InstallMode = "scoop"
)

func NormalizeVersion(version string) string {
	normalized := strings.TrimSpace(version)
	if normalized == "" || strings.EqualFold(normalized, "latest") {
		return "latest"
	}
	if strings.HasPrefix(strings.ToLower(normalized), "v") {
		return "v" + normalized[1:]
	}
	return "v" + normalized
}

func DetectInstallMode(executablePath string) InstallMode {
	normalizedPath := strings.ToLower(filepath.ToSlash(strings.ReplaceAll(executablePath, "\\", "/")))

	switch {
	case strings.Contains(normalizedPath, "/cellar/gig-cli/"):
		return ModeHomebrew
	case strings.Contains(normalizedPath, "/scoop/apps/gig-cli/"):
		return ModeScoop
	default:
		return ModeDirect
	}
}
