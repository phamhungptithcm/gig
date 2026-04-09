package conflict

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gig/internal/repo"
	"gig/internal/scm"
	"gig/internal/ticket"
)

var ErrNoConflict = errors.New("no active conflict state")

type adapterRegistry interface {
	All() []scm.Adapter
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterRegistry
	parser     ticket.Parser
}

func NewService(discoverer repo.Discoverer, adapters adapterRegistry, parser ticket.Parser) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
		parser:     parser,
	}
}

func (s *Service) Status(ctx context.Context, path, scopeTicketID string) (Status, error) {
	scopeTicketID = normalizeScopeTicket(scopeTicketID)
	if scopeTicketID != "" {
		if err := s.parser.Validate(scopeTicketID); err != nil {
			return Status{}, err
		}
	}

	repository, provider, err := s.locateConflictRepository(ctx, path)
	if err != nil {
		return Status{}, err
	}

	operation, active, err := provider.ConflictState(ctx, repository.Root)
	if err != nil {
		return Status{}, err
	}
	if !active {
		return Status{}, ErrNoConflict
	}

	conflictFiles, err := provider.ConflictFiles(ctx, repository.Root)
	if err != nil {
		return Status{}, err
	}

	files := make([]FileStatus, 0, len(conflictFiles))
	resolvable := 0
	unsupported := 0
	for _, file := range conflictFiles {
		status := s.classifyFile(ctx, repository.Root, provider, operation, file, scopeTicketID)
		if status.Supported {
			resolvable++
		} else {
			unsupported++
		}
		files = append(files, status)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	suggestedNext := "Resolve conflicts manually in your editor, then continue the Git operation."
	if resolvable > 0 {
		suggestedNext = fmt.Sprintf("Run `gig resolve start --path %s` to walk the supported text conflicts.", repository.Root)
	}

	return Status{
		Repository:       repository,
		Operation:        operation,
		Files:            files,
		ResolvableFiles:  resolvable,
		UnsupportedFiles: unsupported,
		ScopeTicketID:    scopeTicketID,
		SuggestedNext:    suggestedNext,
	}, nil
}

func (s *Service) LoadActiveConflict(ctx context.Context, path, currentFile string, currentBlock int, scopeTicketID string) (Session, *ActiveConflict, error) {
	status, err := s.Status(ctx, path, scopeTicketID)
	if err != nil {
		return Session{}, nil, err
	}

	repository, _, err := s.locateConflictRepository(ctx, path)
	if err != nil {
		return Session{}, nil, err
	}

	choices := supportedFiles(status.Files)
	if len(choices) == 0 {
		return Session{Status: status}, nil, nil
	}

	selectedFile := currentFile
	if selectedFile == "" || !containsFile(choices, selectedFile) {
		selectedFile = choices[0]
		currentBlock = 0
	}

	fileStatus := findFileStatus(status.Files, selectedFile)
	if fileStatus == nil {
		selectedFile = choices[0]
		fileStatus = findFileStatus(status.Files, selectedFile)
		currentBlock = 0
	}

	parsed, err := s.parseWorkingFile(repository.Root, selectedFile, status.Operation)
	if err != nil {
		return Session{}, nil, err
	}
	if len(parsed.Blocks) == 0 {
		nextFile := nextFileWithBlocks(s, repository, status, choices, selectedFile)
		if nextFile == "" {
			return Session{Status: status, CurrentFile: selectedFile, CurrentBlock: 0}, nil, nil
		}
		selectedFile = nextFile
		currentBlock = 0
		fileStatus = findFileStatus(status.Files, selectedFile)
		parsed, err = s.parseWorkingFile(repository.Root, selectedFile, status.Operation)
		if err != nil {
			return Session{}, nil, err
		}
	}

	if len(parsed.Blocks) == 0 {
		return Session{Status: status, CurrentFile: selectedFile, CurrentBlock: 0}, nil, nil
	}

	if currentBlock < 0 || currentBlock >= len(parsed.Blocks) {
		currentBlock = 0
	}

	block := parsed.Blocks[currentBlock]
	risk := assessRisk(selectedFile, block)
	scopeWarnings := deriveScopeWarnings(scopeTicketID, block, status.Operation)

	active := &ActiveConflict{
		Repository:    repository,
		Operation:     status.Operation,
		File:          *fileStatus,
		Block:         block,
		Risk:          risk,
		ScopeTicketID: status.ScopeTicketID,
		ScopeWarnings: scopeWarnings,
	}

	return Session{
		Status:       status,
		CurrentFile:  selectedFile,
		CurrentBlock: currentBlock,
	}, active, nil
}

func (s *Service) ApplyResolution(ctx context.Context, repoRoot, path string, blockIndex int, operation scm.ConflictOperationState, choice ResolutionChoice) error {
	parsed, err := s.parseWorkingFile(repoRoot, path, operation)
	if err != nil {
		return err
	}

	updated, err := ApplyResolution(parsed, blockIndex, choice)
	if err != nil {
		return err
	}

	return writeWorkingFile(filepath.Join(repoRoot, path), updated)
}

func (s *Service) StageFile(ctx context.Context, repoRoot, path string) error {
	provider, err := s.providerForRepoRoot(ctx, repoRoot)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, path))
	if err != nil {
		return err
	}
	if ContainsConflictMarkers(content) {
		return fmt.Errorf("file %s still contains conflict markers", path)
	}

	return provider.StageConflictFile(ctx, repoRoot, path)
}

func (s *Service) parseWorkingFile(repoRoot, path string, operation scm.ConflictOperationState) (ParsedFile, error) {
	content, err := os.ReadFile(filepath.Join(repoRoot, path))
	if err != nil {
		return ParsedFile{}, err
	}

	parsed := parseFile(content)
	for i := range parsed.Blocks {
		parsed.Blocks[i].CurrentRef = operation.CurrentSide
		parsed.Blocks[i].IncomingRef = operation.IncomingSide
	}
	for i := range parsed.Segments {
		if parsed.Segments[i].Block == nil {
			continue
		}
		index := parsed.Segments[i].Block.Index
		parsed.Segments[i].Block = &parsed.Blocks[index]
	}

	return parsed, nil
}

func (s *Service) classifyFile(_ context.Context, repoRoot string, _ scm.ConflictProvider, operation scm.ConflictOperationState, file scm.ConflictFile, scopeTicketID string) FileStatus {
	status := FileStatus{
		Path:         file.Path,
		ConflictCode: file.ConflictCode,
	}

	switch file.ConflictCode {
	case "UU":
	default:
		status.UnsupportedReason = fmt.Sprintf("conflict type %s is not supported by `gig resolve start` yet", file.ConflictCode)
		status.Warnings = []string{"Resolve this file manually, then stage it yourself."}
		return status
	}

	parsed, err := s.parseWorkingFile(repoRoot, file.Path, operation)
	if err != nil {
		status.UnsupportedReason = err.Error()
		status.Warnings = []string{"Resolve this file manually, then stage it yourself."}
		return status
	}

	status.BlockCount = len(parsed.Blocks)
	if len(parsed.Blocks) == 0 {
		status.Supported = true
		status.Warnings = []string{"No inline conflict markers remain in the working tree. Stage the file if the result is correct."}
		return status
	}

	status.Supported = true
	if scopeTicketID != "" {
		for _, warning := range deriveScopeWarnings(scopeTicketID, parsed.Blocks[0], operation) {
			status.Warnings = append(status.Warnings, warning)
		}
	}

	risk := assessRisk(file.Path, parsed.Blocks[0])
	if risk.Severity != "" && risk.Summary != "" {
		status.Warnings = append(status.Warnings, risk.Summary)
	}

	return status
}

func (s *Service) locateConflictRepository(ctx context.Context, path string) (scm.Repository, scm.ConflictProvider, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "."
	}

	for _, adapter := range s.adapters.All() {
		provider, ok := adapter.(scm.ConflictProvider)
		if !ok {
			continue
		}
		root, found, err := adapter.DetectRoot(path)
		if err != nil {
			return scm.Repository{}, nil, err
		}
		if !found {
			continue
		}

		repository := scm.Repository{
			Name: filepath.Base(root),
			Root: root,
			Type: adapter.Type(),
		}
		repository.CurrentBranch, _ = adapter.CurrentBranch(ctx, root)
		return repository, provider, nil
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return scm.Repository{}, nil, err
	}
	if len(repositories) != 1 {
		return scm.Repository{}, nil, fmt.Errorf("resolve requires a path inside one Git repository")
	}

	repository := repositories[0]
	provider, ok := providerFromAdapters(s.adapters.All(), repository.Type)
	if !ok {
		return scm.Repository{}, nil, fmt.Errorf("resolve is only supported for Git repositories")
	}
	return repository, provider, nil
}

func (s *Service) providerForRepoRoot(ctx context.Context, repoRoot string) (scm.ConflictProvider, error) {
	repository, provider, err := s.locateConflictRepository(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	if repository.Root != repoRoot {
		return nil, fmt.Errorf("repository %s is not active", repoRoot)
	}
	return provider, nil
}

func providerFromAdapters(adapters []scm.Adapter, repoType scm.Type) (scm.ConflictProvider, bool) {
	for _, adapter := range adapters {
		if adapter.Type() != repoType {
			continue
		}
		provider, ok := adapter.(scm.ConflictProvider)
		if ok {
			return provider, true
		}
	}
	return nil, false
}

func supportedFiles(files []FileStatus) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if !file.Supported {
			continue
		}
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}

func findFileStatus(files []FileStatus, path string) *FileStatus {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func containsFile(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}

func nextFileWithBlocks(s *Service, repository scm.Repository, status Status, files []string, currentFile string) string {
	start := 0
	for i, path := range files {
		if path == currentFile {
			start = i + 1
			break
		}
	}

	for i := 0; i < len(files); i++ {
		path := files[(start+i)%len(files)]
		parsed, err := s.parseWorkingFile(repository.Root, path, status.Operation)
		if err != nil {
			continue
		}
		if len(parsed.Blocks) > 0 {
			return path
		}
	}
	return ""
}

func writeWorkingFile(path string, content []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
	}
	return os.WriteFile(path, content, mode)
}

func normalizeScopeTicket(ticketID string) string {
	return strings.ToUpper(strings.TrimSpace(ticketID))
}
