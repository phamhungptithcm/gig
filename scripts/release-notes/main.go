package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	releasenotes "gig/internal/releasenotes"
)

var dateTagPattern = regexp.MustCompile(`^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	previousTag, currentTag, err := parseArgs(args)
	if err != nil {
		return err
	}

	if previousTag != "" {
		if _, err := runGit("rev-parse", "-q", "--verify", "refs/tags/"+previousTag); err != nil {
			return fmt.Errorf("unknown previous tag: %s", previousTag)
		}
	}

	subjects := []string{}
	if !shouldRenderInitialRelease(previousTag) {
		logRange := "HEAD"
		if previousTag != "" {
			logRange = previousTag + "..HEAD"
		}

		output, err := runGit("log", "--no-merges", "--reverse", "--format=%s", logRange)
		if err != nil {
			return err
		}
		subjects = splitLines(output)
	}

	fmt.Print(releasenotes.GenerateMarkdown(releasenotes.Input{
		RepositoryName: "gig",
		PreviousTag:    previousTag,
		CurrentTag:     currentTag,
		CompareURL:     compareURL(previousTag, currentTag),
		Subjects:       subjects,
	}))
	return nil
}

func parseArgs(args []string) (string, string, error) {
	switch len(args) {
	case 1:
		return "", strings.TrimSpace(args[0]), nil
	case 2:
		return strings.TrimSpace(args[0]), strings.TrimSpace(args[1]), nil
	default:
		return "", "", fmt.Errorf("usage: release-notes.sh [previous-tag] <current-tag>")
	}
}

func shouldRenderInitialRelease(previousTag string) bool {
	previousTag = strings.TrimSpace(previousTag)
	return previousTag == "" || !dateTagPattern.MatchString(previousTag)
}

func compareURL(previousTag, currentTag string) string {
	if previousTag == "" || currentTag == "" {
		return ""
	}

	remoteURL, err := runGit("remote", "get-url", "origin")
	if err != nil {
		return ""
	}

	baseURL, ok := normalizeGitHubURL(strings.TrimSpace(remoteURL))
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/compare/%s...%s", baseURL, previousTag, currentTag)
}

func normalizeGitHubURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	switch {
	case strings.HasPrefix(raw, "git@github.com:"):
		return "https://github.com/" + strings.TrimPrefix(raw, "git@github.com:"), true
	case strings.HasPrefix(raw, "ssh://git@github.com/"):
		return "https://github.com/" + strings.TrimPrefix(raw, "ssh://git@github.com/"), true
	case strings.HasPrefix(raw, "https://github.com/"):
		return raw, true
	case strings.HasPrefix(raw, "http://github.com/"):
		return "https://github.com/" + strings.TrimPrefix(raw, "http://github.com/"), true
	default:
		return "", false
	}
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%v: %s", err, message)
	}
	return string(output), nil
}

func splitLines(value string) []string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}
