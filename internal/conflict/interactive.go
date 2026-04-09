package conflict

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gig/internal/scm"

	"golang.org/x/term"
)

type InteractiveOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type undoEntry struct {
	path    string
	content []byte
}

func (s *Service) RunInteractive(ctx context.Context, path, scopeTicketID string, options InteractiveOptions) error {
	if options.Stdin == nil {
		options.Stdin = os.Stdin
	}
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}

	reader := newCommandReader(options.Stdin)
	defer reader.Close()

	var undoStack []undoEntry
	showDetails := false
	showRisk := true

	currentFile := ""
	currentBlock := 0

	for {
		session, active, err := s.LoadActiveConflict(ctx, path, currentFile, currentBlock, scopeTicketID)
		if err != nil {
			return err
		}
		if active == nil {
			return renderCompletion(options.Stdout, session.Status)
		}

		currentFile = session.CurrentFile
		currentBlock = session.CurrentBlock

		if err := renderInteractiveView(options.Stdout, session, active, showDetails, showRisk, reader.IsTTY()); err != nil {
			return err
		}

		command, err := reader.ReadCommand()
		if err != nil {
			return err
		}

		switch command {
		case "1":
			if err := pushUndo(&undoStack, active.Repository.Root, active.File.Path); err != nil {
				return err
			}
			if err := s.ApplyResolution(ctx, active.Repository.Root, active.File.Path, active.Block.Index, active.Operation, ResolutionCurrent); err != nil {
				return err
			}
		case "2":
			if err := pushUndo(&undoStack, active.Repository.Root, active.File.Path); err != nil {
				return err
			}
			if err := s.ApplyResolution(ctx, active.Repository.Root, active.File.Path, active.Block.Index, active.Operation, ResolutionIncoming); err != nil {
				return err
			}
		case "3":
			if err := pushUndo(&undoStack, active.Repository.Root, active.File.Path); err != nil {
				return err
			}
			if err := s.ApplyResolution(ctx, active.Repository.Root, active.File.Path, active.Block.Index, active.Operation, ResolutionBothCurrentFirst); err != nil {
				return err
			}
		case "4":
			if err := pushUndo(&undoStack, active.Repository.Root, active.File.Path); err != nil {
				return err
			}
			if err := s.ApplyResolution(ctx, active.Repository.Root, active.File.Path, active.Block.Index, active.Operation, ResolutionBothIncomingFirst); err != nil {
				return err
			}
		case "e":
			if err := pushUndo(&undoStack, active.Repository.Root, active.File.Path); err != nil {
				return err
			}
			if err := openEditor(reader, filepath.Join(active.Repository.Root, active.File.Path)); err != nil {
				return err
			}
		case "n":
			currentBlock++
		case "p":
			if currentBlock > 0 {
				currentBlock--
			}
		case "N":
			currentFile, currentBlock = jumpFile(session.Status.Files, currentFile, 1)
		case "P":
			currentFile, currentBlock = jumpFile(session.Status.Files, currentFile, -1)
		case "d":
			showDetails = !showDetails
		case "r":
			showRisk = !showRisk
		case "u":
			entry, ok := popUndo(&undoStack)
			if !ok {
				continue
			}
			if err := writeWorkingFile(filepath.Join(active.Repository.Root, entry.path), entry.content); err != nil {
				return err
			}
			currentFile = entry.path
			currentBlock = 0
		case "s":
			if err := s.StageFile(ctx, active.Repository.Root, active.File.Path); err != nil {
				if _, writeErr := fmt.Fprintf(options.Stderr, "stage failed: %v\n", err); writeErr != nil {
					return writeErr
				}
			}
		case "q":
			if _, err := fmt.Fprintln(options.Stdout, "Resolver closed without continuing the Git operation."); err != nil {
				return err
			}
			return nil
		default:
			continue
		}
	}
}

func renderInteractiveView(w io.Writer, session Session, active *ActiveConflict, showDetails, showRisk, clearScreen bool) error {
	if clearScreen {
		if _, err := io.WriteString(w, "\033[H\033[2J"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "gig resolve\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Repo: %s\n", active.Repository.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Operation: %s\n", active.Operation.Type); err != nil {
		return err
	}
	if active.Operation.SequenceBranch != "" {
		if _, err := fmt.Fprintf(w, "Sequence: %s\n", active.Operation.SequenceBranch); err != nil {
			return err
		}
	}
	if active.Operation.TargetBranch != "" {
		if _, err := fmt.Fprintf(w, "Target: %s\n", active.Operation.TargetBranch); err != nil {
			return err
		}
	}

	totalBlocks := active.File.BlockCount
	if totalBlocks < 1 {
		totalBlocks = 1
	}
	if _, err := fmt.Fprintf(w, "File: %s (%d/%d)\n", active.File.Path, active.Block.Index+1, totalBlocks); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "\n%s\n", formatBlockLabel(active.Operation.CurrentSide)); err != nil {
		return err
	}
	if _, err := writePreview(w, active.Block.Current); err != nil {
		return err
	}

	if len(active.Block.Base) > 0 {
		if _, err := fmt.Fprintln(w, "\nBase:"); err != nil {
			return err
		}
		if _, err := writePreview(w, active.Block.Base); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n%s\n", formatBlockLabel(active.Operation.IncomingSide)); err != nil {
		return err
	}
	if _, err := writePreview(w, active.Block.Incoming); err != nil {
		return err
	}

	if showRisk {
		if _, err := fmt.Fprintf(w, "\nRisk: %s\n", active.Risk.Summary); err != nil {
			return err
		}
		for _, note := range active.Risk.ReviewNotes {
			if _, err := fmt.Fprintf(w, "  - %s\n", note); err != nil {
				return err
			}
		}
		for _, line := range active.Risk.DuplicateLines {
			if _, err := fmt.Fprintf(w, "  - duplicate candidate: %s\n", line); err != nil {
				return err
			}
		}
	}

	if len(active.ScopeWarnings) > 0 {
		if _, err := fmt.Fprintln(w, "\nWarnings:"); err != nil {
			return err
		}
		for _, warning := range active.ScopeWarnings {
			if _, err := fmt.Fprintf(w, "  - %s\n", warning); err != nil {
				return err
			}
		}
	}

	if showDetails {
		if _, err := fmt.Fprintln(w, "\nFiles:"); err != nil {
			return err
		}
		for _, file := range session.Status.Files {
			marker := " "
			if file.Path == active.File.Path {
				marker = ">"
			}
			if _, err := fmt.Fprintf(w, "%s %s [%s] blocks=%d\n", marker, file.Path, file.ConflictCode, file.BlockCount); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintln(w, "\nKeys: 1 current  2 incoming  3 both  4 both-reverse  e edit  s stage  u undo  n/p block  N/P file  d details  r risk  q quit")
	return err
}

func renderCompletion(w io.Writer, status Status) error {
	if _, err := fmt.Fprintf(w, "All supported conflict blocks are resolved.\n"); err != nil {
		return err
	}
	if status.UnsupportedFiles > 0 {
		if _, err := fmt.Fprintf(w, "%d unsupported file(s) still need manual resolution.\n", status.UnsupportedFiles); err != nil {
			return err
		}
	}
	if status.Operation.ContinuationCommand != "" {
		if _, err := fmt.Fprintf(w, "Next command: %s\n", status.Operation.ContinuationCommand); err != nil {
			return err
		}
	}
	return nil
}

func writePreview(w io.Writer, content []byte) (int, error) {
	text := strings.TrimRight(string(content), "\n")
	if text == "" {
		return fmt.Fprintln(w, "  (empty)")
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 12 {
		lines = lines[:12]
		lines = append(lines, "... (truncated)")
	}

	written := 0
	for _, line := range lines {
		n, err := fmt.Fprintf(w, "  %s\n", line)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func formatBlockLabel(side scm.ConflictSide) string {
	parts := make([]string, 0, 5)
	parts = append(parts, side.Label)
	if side.Branch != "" {
		parts = append(parts, side.Branch)
	}
	if short := side.ShortHash(); short != "" {
		parts = append(parts, short)
	}
	if side.Subject != "" {
		parts = append(parts, side.Subject)
	}
	if len(side.TicketIDs) > 0 {
		parts = append(parts, strings.Join(side.TicketIDs, ","))
	}
	return strings.Join(parts, " | ")
}

type commandReader struct {
	input  io.Reader
	buffer *bufio.Reader
	state  *term.State
	fd     int
	isTTY  bool
}

func newCommandReader(input io.Reader) *commandReader {
	reader := &commandReader{
		input:  input,
		buffer: bufio.NewReader(input),
		fd:     -1,
	}

	fdProvider, ok := input.(interface{ Fd() uintptr })
	if !ok {
		return reader
	}

	fd := int(fdProvider.Fd())
	if !term.IsTerminal(fd) {
		return reader
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return reader
	}

	reader.fd = fd
	reader.state = state
	reader.isTTY = true
	return reader
}

func (r *commandReader) ReadCommand() (string, error) {
	if !r.isTTY {
		line, err := r.buffer.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		if err == io.EOF && line == "" {
			return "q", nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return "", nil
		}
		return string(line[0]), nil
	}

	for {
		b, err := r.buffer.ReadByte()
		if err != nil {
			return "", err
		}
		switch b {
		case '\r', '\n':
			continue
		case 3:
			return "q", nil
		default:
			return string(b), nil
		}
	}
}

func (r *commandReader) Close() {
	if r.state != nil && r.fd >= 0 {
		_ = term.Restore(r.fd, r.state)
	}
}

func (r *commandReader) IsTTY() bool {
	return r.isTTY
}

func openEditor(reader *commandReader, path string) error {
	if reader.state != nil && reader.fd >= 0 {
		if err := term.Restore(reader.fd, reader.state); err != nil {
			return err
		}
		defer func() {
			reader.state, _ = term.MakeRaw(reader.fd)
		}()
	}

	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}

	fields := strings.Fields(editor)
	command := exec.Command(fields[0], append(fields[1:], path)...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func pushUndo(stack *[]undoEntry, repoRoot, path string) error {
	content, err := os.ReadFile(filepath.Join(repoRoot, path))
	if err != nil {
		return err
	}

	*stack = append(*stack, undoEntry{
		path:    path,
		content: content,
	})
	return nil
}

func popUndo(stack *[]undoEntry) (undoEntry, bool) {
	if len(*stack) == 0 {
		return undoEntry{}, false
	}

	index := len(*stack) - 1
	entry := (*stack)[index]
	*stack = (*stack)[:index]
	return entry, true
}

func jumpFile(files []FileStatus, currentFile string, direction int) (string, int) {
	supported := supportedFiles(files)
	if len(supported) == 0 {
		return currentFile, 0
	}

	currentIndex := 0
	for i, path := range supported {
		if path == currentFile {
			currentIndex = i
			break
		}
	}

	currentIndex = (currentIndex + direction + len(supported)) % len(supported)
	return supported[currentIndex], 0
}
