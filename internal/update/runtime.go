package update

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type InstallMode string

const (
	ModeDirect   InstallMode = "direct"
	ModeNPM      InstallMode = "npm"
	ModeHomebrew InstallMode = "homebrew"
	ModeScoop    InstallMode = "scoop"
)

const DefaultNPMPackageName = "@hunpeolabs/gig"

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

func NormalizeNPMVersion(version string) (string, error) {
	normalized := NormalizeVersion(version)
	if normalized == "latest" {
		return normalized, nil
	}

	parts := strings.Split(strings.TrimPrefix(normalized, "v"), ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("release version %q must use vYYYY.MM.DD", normalized)
	}

	npmParts := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return "", fmt.Errorf("release version %q must use vYYYY.MM.DD", normalized)
		}
		npmParts = append(npmParts, strconv.Itoa(value))
	}

	return strings.Join(npmParts, "."), nil
}

func DetectInstallMode(executablePath string, lookupEnv func(string) (string, bool)) InstallMode {
	if lookupEnv != nil {
		if value, ok := lookupEnv("GIG_INSTALL_MODE"); ok {
			switch strings.ToLower(strings.TrimSpace(value)) {
			case string(ModeNPM):
				return ModeNPM
			case string(ModeDirect):
				return ModeDirect
			}
		}
	}

	normalizedPath := strings.ToLower(filepath.ToSlash(strings.ReplaceAll(executablePath, "\\", "/")))

	switch {
	case strings.Contains(normalizedPath, "/node_modules/"):
		return ModeNPM
	case strings.Contains(normalizedPath, "/cellar/gig-cli/"):
		return ModeHomebrew
	case strings.Contains(normalizedPath, "/scoop/apps/gig-cli/"):
		return ModeScoop
	default:
		return ModeDirect
	}
}

func ResolveNPMPackageName(lookupEnv func(string) (string, bool)) string {
	if lookupEnv != nil {
		if value, ok := lookupEnv("GIG_NPM_PACKAGE_NAME"); ok {
			if normalized := strings.TrimSpace(value); normalized != "" {
				return normalized
			}
		}
	}

	return DefaultNPMPackageName
}
