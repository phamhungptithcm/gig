package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	updatesvc "gig/internal/update"
)

func (a *App) runUpdate(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printUpdateUsage()
		return 0
	}

	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	versionFlag := fs.String("version", "latest", "Install the latest release or a specific tag")
	repoFlag := fs.String("repo", "", "GitHub repo that hosts gig releases")
	installDirFlag := fs.String("install-dir", "", "Override the install directory for a direct install")

	if err := fs.Parse(args); err != nil {
		a.printUpdateUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(a.stderr, "update accepts at most one positional <version> argument")
		a.printUpdateUsage()
		return usageExitCode
	}

	version := updatesvc.NormalizeVersion(*versionFlag)
	if fs.NArg() == 1 {
		if version != "latest" {
			fmt.Fprintln(a.stderr, "update accepts either --version or a positional <version>, not both")
			a.printUpdateUsage()
			return usageExitCode
		}
		version = updatesvc.NormalizeVersion(fs.Arg(0))
	}

	repoName := strings.TrimSpace(*repoFlag)
	if repoName == "" {
		repoName = strings.TrimSpace(os.Getenv("GIG_REPO"))
	}
	if repoName == "" {
		repoName = "phamhungptithcm/gig"
	}

	executablePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	resolvedExecutablePath := executablePath
	if linkedPath, linkErr := filepath.EvalSymlinks(executablePath); linkErr == nil {
		resolvedExecutablePath = linkedPath
	}

	installDir := strings.TrimSpace(*installDirFlag)
	installMode := updatesvc.DetectInstallMode(resolvedExecutablePath, os.LookupEnv)
	if installDir != "" {
		installMode = updatesvc.ModeDirect
	} else {
		installDir = filepath.Dir(resolvedExecutablePath)
	}

	switch installMode {
	case updatesvc.ModeNPM:
		fmt.Fprintf(a.stdout, "Detected an npm-managed install at %s\n", resolvedExecutablePath)
		return a.runNPMUpdate(ctx, updatesvc.ResolveNPMPackageName(os.LookupEnv), version)
	case updatesvc.ModeHomebrew:
		fmt.Fprintf(a.stderr, "update failed: Homebrew installs are no longer published for gig. Reinstall with `npm install -g %s` or use the direct installer.\n", updatesvc.DefaultNPMPackageName)
		return 1
	case updatesvc.ModeScoop:
		fmt.Fprintf(a.stderr, "update failed: Scoop installs are no longer published for gig. Reinstall with `npm install -g %s` or use the direct installer.\n", updatesvc.DefaultNPMPackageName)
		return 1
	default:
		if runtime.GOOS == "windows" {
			return a.runWindowsInstallerUpdate(ctx, repoName, version, installDir)
		}
		return a.runPOSIXInstallerUpdate(ctx, repoName, version, installDir)
	}
}

func (a *App) runNPMUpdate(ctx context.Context, packageName, releaseVersion string) int {
	npmVersion, err := updatesvc.NormalizeNPMVersion(releaseVersion)
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	spec := packageName + "@latest"
	if npmVersion != "latest" {
		spec = packageName + "@" + npmVersion
	}

	if runtime.GOOS == "windows" {
		return a.runWindowsNPMUpdate(ctx, spec)
	}

	return a.runExternalCommand(ctx, "npm", []string{"install", "-g", spec})
}

func (a *App) runExternalCommand(ctx context.Context, name string, args []string) int {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runPOSIXInstallerUpdate(ctx context.Context, repoName, version, installDir string) int {
	installerURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/scripts/install.sh", repoName)
	command := fmt.Sprintf(
		"if command -v curl >/dev/null 2>&1; then curl -fsSL %s | sh; elif command -v wget >/dev/null 2>&1; then wget -qO- %s | sh; else echo 'curl or wget is required to update gig.' >&2; exit 1; fi",
		shellSingleQuote(installerURL),
		shellSingleQuote(installerURL),
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr
	cmd.Env = append(os.Environ(),
		"GIG_REPO="+repoName,
		"GIG_VERSION="+version,
		"GIG_INSTALL_DIR="+installDir,
	)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runWindowsInstallerUpdate(ctx context.Context, repoName, version, installDir string) int {
	installerURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/scripts/install.ps1", repoName)

	scriptFile, err := os.CreateTemp("", "gig-update-*.ps1")
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	scriptBody := fmt.Sprintf(`$ErrorActionPreference = "Stop"
$installer = [ScriptBlock]::Create((Invoke-RestMethod -Uri '%s'))
& $installer -Repo '%s' -Version '%s' -InstallDir '%s' -WaitForPid %d
Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
`,
		powerShellSingleQuote(installerURL),
		powerShellSingleQuote(repoName),
		powerShellSingleQuote(version),
		powerShellSingleQuote(installDir),
		os.Getpid(),
	)

	if _, err := scriptFile.WriteString(scriptBody); err != nil {
		scriptFile.Close()
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}
	if err := scriptFile.Close(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	command := fmt.Sprintf(
		"Start-Process powershell -WindowStyle Hidden -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File','%s')",
		powerShellSingleQuote(scriptFile.Name()),
	)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(a.stdout, "gig update started in the background.")
	fmt.Fprintln(a.stdout, "Open a new terminal in a few seconds, then run: gig version")
	return 0
}

func (a *App) runWindowsNPMUpdate(ctx context.Context, packageSpec string) int {
	scriptFile, err := os.CreateTemp("", "gig-update-npm-*.ps1")
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	scriptBody := fmt.Sprintf(`$ErrorActionPreference = "Stop"
for ($attempt = 0; $attempt -lt 240; $attempt++) {
	if (-not (Get-Process -Id %d -ErrorAction SilentlyContinue)) {
		break
	}
	Start-Sleep -Milliseconds 500
}
npm.cmd install -g '%s'
Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
`,
		os.Getpid(),
		powerShellSingleQuote(packageSpec),
	)

	if _, err := scriptFile.WriteString(scriptBody); err != nil {
		scriptFile.Close()
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}
	if err := scriptFile.Close(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	command := fmt.Sprintf(
		"Start-Process powershell -WindowStyle Hidden -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File','%s')",
		powerShellSingleQuote(scriptFile.Name()),
	)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(a.stdout, "gig update started in the background.")
	fmt.Fprintln(a.stdout, "Open a new terminal in a few seconds, then run: gig version")
	return 0
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
