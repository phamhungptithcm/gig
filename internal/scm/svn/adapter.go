package svn

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

type Adapter struct {
	parser ticket.Parser
}

type infoDocument struct {
	Entries []infoEntry `xml:"entry"`
}

type infoEntry struct {
	URL         string `xml:"url"`
	RelativeURL string `xml:"relative-url"`
	Repository  struct {
		Root string `xml:"root"`
	} `xml:"repository"`
}

type logDocument struct {
	Entries []logEntry `xml:"logentry"`
}

type logEntry struct {
	Revision string    `xml:"revision,attr"`
	Message  string    `xml:"msg"`
	Paths    []logPath `xml:"paths>path"`
}

type logPath struct {
	Value string `xml:",chardata"`
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{parser: parser}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeSVN
}

func (a *Adapter) DetectRoot(path string) (string, bool, error) {
	start, err := normalizePath(path)
	if err != nil {
		return "", false, err
	}

	for {
		ok, err := a.IsRepository(start)
		if err != nil {
			return "", false, err
		}
		if ok {
			return start, true, nil
		}

		parent := filepath.Dir(start)
		if parent == start {
			return "", false, nil
		}
		start = parent
	}
}

func (a *Adapter) IsRepository(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, ".svn"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func (a *Adapter) CurrentBranch(ctx context.Context, repoRoot string) (string, error) {
	info, err := a.readInfo(ctx, repoRoot)
	if err != nil {
		return "", err
	}

	return branchNameFromInfo(info), nil
}

func (a *Adapter) SearchCommits(ctx context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if err := a.parser.Validate(query.TicketID); err != nil {
		return nil, err
	}

	target := repoRoot
	defaultBranch := ""
	if strings.TrimSpace(query.Branch) != "" {
		info, err := a.readInfo(ctx, repoRoot)
		if err != nil {
			return nil, err
		}

		target, defaultBranch, err = resolveBranchTarget(info, query.Branch)
		if err != nil {
			return nil, err
		}
	}

	output, err := a.runSVN(ctx, "log", "--xml", "--verbose", target)
	if err != nil {
		return nil, err
	}

	entries, err := parseLogEntries(output)
	if err != nil {
		return nil, err
	}

	return buildCommits(entries, a.parser, query.TicketID, defaultBranch), nil
}

func (a *Adapter) CompareBranches(context.Context, string, scm.CompareQuery) (scm.CompareResult, error) {
	return scm.CompareResult{}, scm.ErrUnsupported
}

func (a *Adapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func (a *Adapter) RefExists(ctx context.Context, repoRoot, ref string) (bool, error) {
	info, err := a.readInfo(ctx, repoRoot)
	if err != nil {
		return false, err
	}

	target, _, err := resolveBranchTarget(info, ref)
	if err != nil {
		return false, err
	}

	if _, err := a.runSVN(ctx, "info", "--xml", target); err != nil {
		if strings.Contains(err.Error(), "not a working copy") || strings.Contains(err.Error(), "non-existent") || strings.Contains(err.Error(), "E160013") || strings.Contains(err.Error(), "E200009") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
	filesByCommit := make(map[string][]string, len(hashes))
	seen := make(map[string]struct{}, len(hashes))

	for _, hash := range hashes {
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		revision := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(hash), "r"), "R")
		if revision == "" {
			filesByCommit[hash] = nil
			continue
		}

		output, err := a.runSVN(ctx, "log", "--xml", "--verbose", "-r", revision, repoRoot)
		if err != nil {
			return nil, err
		}

		entries, err := parseLogEntries(output)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			filesByCommit[hash] = nil
			continue
		}

		filesByCommit[hash] = changedFiles(entries[0].Paths)
	}

	return filesByCommit, nil
}

func buildCommits(entries []logEntry, parser ticket.Parser, ticketID, defaultBranch string) []scm.Commit {
	commits := make([]scm.Commit, 0, len(entries))
	for _, entry := range entries {
		message := strings.TrimSpace(entry.Message)
		if !parser.Matches(ticketID, message) {
			continue
		}

		branches := collectBranches(entry.Paths, defaultBranch)
		commits = append(commits, scm.Commit{
			Hash:     revisionHash(entry.Revision),
			Subject:  commitSubject(message),
			Branches: branches,
		})
	}

	return commits
}

func changedFiles(paths []logPath) []string {
	files := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))

	for _, changedPath := range paths {
		normalized := normalizeChangedPath(changedPath.Value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		files = append(files, normalized)
	}

	sort.Strings(files)
	return files
}

func collectBranches(paths []logPath, defaultBranch string) []string {
	if strings.TrimSpace(defaultBranch) != "" {
		return []string{defaultBranch}
	}

	branches := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, changedPath := range paths {
		branch := branchFromChangedPath(changedPath.Value)
		if branch == "" {
			continue
		}
		if _, ok := seen[branch]; ok {
			continue
		}
		seen[branch] = struct{}{}
		branches = append(branches, branch)
	}

	sort.Strings(branches)
	return branches
}

func resolveBranchTarget(info infoEntry, branch string) (string, string, error) {
	relativePath, display := resolveBranchPath(strings.TrimSpace(branch), info.RelativeURL)
	if relativePath == "" {
		return "", "", fmt.Errorf("branch is required")
	}
	if strings.TrimSpace(info.Repository.Root) == "" {
		return "", "", fmt.Errorf("svn repository root URL is not available")
	}

	return joinURL(info.Repository.Root, relativePath), display, nil
}

func resolveBranchPath(branch, currentRelativeURL string) (string, string) {
	branch = strings.TrimSpace(branch)
	branch = strings.TrimPrefix(branch, "^/")
	branch = strings.Trim(branch, "/")
	if branch == "" {
		return "", ""
	}

	switch {
	case branch == "trunk":
		return branch, "trunk"
	case strings.HasPrefix(branch, "branches/"), strings.HasPrefix(branch, "tags/"):
		return branch, displayBranchName(branch)
	}

	current := strings.Trim(strings.TrimPrefix(strings.TrimSpace(currentRelativeURL), "^/"), "/")
	if current == "" || current == "trunk" || strings.HasPrefix(current, "trunk/") || strings.HasPrefix(current, "branches/") || strings.HasPrefix(current, "tags/") {
		return pathpkg.Join("branches", branch), branch
	}

	return branch, displayBranchName(branch)
}

func parseLogEntries(content string) ([]logEntry, error) {
	var document logDocument
	if err := xml.Unmarshal([]byte(content), &document); err != nil {
		return nil, fmt.Errorf("parse svn log xml: %w", err)
	}
	return document.Entries, nil
}

func (a *Adapter) readInfo(ctx context.Context, target string) (infoEntry, error) {
	output, err := a.runSVN(ctx, "info", "--xml", target)
	if err != nil {
		return infoEntry{}, err
	}

	var document infoDocument
	if err := xml.Unmarshal([]byte(output), &document); err != nil {
		return infoEntry{}, fmt.Errorf("parse svn info xml: %w", err)
	}
	if len(document.Entries) == 0 {
		return infoEntry{}, fmt.Errorf("svn info returned no entries for %s", target)
	}

	return document.Entries[0], nil
}

func revisionHash(revision string) string {
	revision = strings.TrimSpace(revision)
	revision = strings.TrimPrefix(strings.TrimPrefix(revision, "r"), "R")
	if revision == "" {
		return ""
	}
	return "r" + revision
}

func commitSubject(message string) string {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "(no message)"
}

func branchFromChangedPath(repoPath string) string {
	root := branchRoot(repoPath)
	if root == "" {
		return ""
	}
	return displayBranchName(root)
}

func normalizeChangedPath(repoPath string) string {
	cleaned := strings.TrimPrefix(pathpkg.Clean("/"+strings.TrimSpace(repoPath)), "/")
	if cleaned == "" || cleaned == "." {
		return ""
	}

	root := branchRoot(repoPath)
	if root == "" {
		return cleaned
	}
	if cleaned == root {
		return ""
	}
	if strings.HasPrefix(cleaned, root+"/") {
		return strings.TrimPrefix(cleaned, root+"/")
	}

	return cleaned
}

func branchRoot(repoPath string) string {
	cleaned := strings.TrimPrefix(pathpkg.Clean("/"+strings.TrimSpace(repoPath)), "/")
	if cleaned == "" || cleaned == "." {
		return ""
	}

	parts := strings.Split(cleaned, "/")
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "trunk":
		return "trunk"
	case "branches", "tags":
		if len(parts) >= 2 && parts[1] != "" {
			return pathpkg.Join(parts[0], parts[1])
		}
	}

	return parts[0]
}

func branchNameFromInfo(info infoEntry) string {
	if branch := displayBranchName(info.RelativeURL); branch != "" {
		return branch
	}

	urlPath := strings.TrimSpace(info.URL)
	repositoryRoot := strings.TrimSpace(info.Repository.Root)
	if repositoryRoot != "" && strings.HasPrefix(urlPath, repositoryRoot) {
		relative := strings.TrimPrefix(urlPath, repositoryRoot)
		if branch := displayBranchName(relative); branch != "" {
			return branch
		}
	}

	return displayBranchName(urlPath)
}

func displayBranchName(location string) string {
	location = strings.Trim(strings.TrimPrefix(strings.TrimSpace(location), "^/"), "/")
	if location == "" {
		return ""
	}

	parts := strings.Split(location, "/")
	switch parts[0] {
	case "trunk":
		return "trunk"
	case "branches":
		if len(parts) >= 2 && parts[1] != "" {
			return parts[1]
		}
	case "tags":
		if len(parts) >= 2 && parts[1] != "" {
			return "tags/" + parts[1]
		}
	}

	return location
}

func joinURL(root, relativePath string) string {
	root = strings.TrimRight(strings.TrimSpace(root), "/")
	relativePath = strings.Trim(strings.TrimSpace(relativePath), "/")
	if relativePath == "" {
		return root
	}
	return root + "/" + relativePath
}

func (a *Adapter) runSVN(ctx context.Context, args ...string) (string, error) {
	if _, err := exec.LookPath("svn"); err != nil {
		return "", fmt.Errorf("svn executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "svn", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("svn %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}

func normalizePath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return absPath, nil
	}

	return filepath.Dir(absPath), nil
}
