package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	inspectsvc "gig/internal/inspect"
	"gig/internal/scm"
	ticketsvc "gig/internal/ticket"

	"golang.org/x/term"
)

func (a *App) commandPromptReader() *bufio.Reader {
	if !a.commandPromptEnabled() {
		return nil
	}
	if reader, ok := a.stdin.(*bufio.Reader); ok {
		return reader
	}
	reader := bufio.NewReader(a.stdin)
	a.stdin = reader
	return reader
}

func (a *App) commandPromptEnabled() bool {
	if file, ok := a.stdin.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}

	type lenReader interface {
		Len() int
	}
	if reader, ok := a.stdin.(lenReader); ok {
		return reader.Len() > 0
	}

	return false
}

func (a *App) promptForRequiredCommandValue(reader *bufio.Reader, label string) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	fmt.Fprintf(a.stderr, "%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return line, nil
}

func (a *App) resolveRequiredTicketArg(reader *bufio.Reader, command string, args []string) (string, error) {
	switch len(args) {
	case 0:
		ticketID, err := a.promptForRequiredCommandValue(reader, "Ticket ID")
		if err != nil {
			return "", fmt.Errorf("%s requires exactly one <ticket-id> argument", command)
		}
		return normalizeTicketID(ticketID), nil
	case 1:
		return normalizeTicketID(args[0]), nil
	default:
		return "", fmt.Errorf("%s requires exactly one <ticket-id> argument", command)
	}
}

func (a *App) resolveTicketIDsWithPrompt(reader *bufio.Reader, ticketID, ticketFile string, parser ticketsvc.Parser) ([]string, string, error) {
	if strings.TrimSpace(ticketID) == "" && strings.TrimSpace(ticketFile) == "" && reader != nil {
		promptedTicketID, err := a.promptForRequiredCommandValue(reader, "Ticket ID")
		if err != nil {
			return nil, "", err
		}
		ticketID = promptedTicketID
	}
	return resolveTicketIDs(ticketID, ticketFile, parser)
}

func (a *App) resolveOperationContextWithPrompt(ctx context.Context, reader *bufio.Reader, command string, runtime commandRuntime, repositories []scm.Repository, envSpec, fromBranch, toBranch string) ([]inspectsvc.Environment, string, string, error) {
	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, envSpec, fromBranch, toBranch)
	if err == nil {
		return environments, resolvedFromBranch, resolvedToBranch, nil
	}
	if reader == nil || !canPromptForPromotionIntent(err, fromBranch, toBranch) {
		return nil, "", "", err
	}

	var topologyErr *topologyResolutionError
	if errors.As(err, &topologyErr) && strings.TrimSpace(topologyErr.Inference.Summary) != "" {
		fmt.Fprintln(a.stderr, topologyErr.Inference.Summary)
	}
	fmt.Fprintf(a.stderr, "Promotion path for %s\n", commandName(command))

	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)
	if fromBranch == "" {
		fromBranch, err = a.promptForRequiredCommandValue(reader, "Source branch")
		if err != nil {
			return nil, "", "", err
		}
	}
	if toBranch == "" {
		toBranch, err = a.promptForRequiredCommandValue(reader, "Target branch")
		if err != nil {
			return nil, "", "", err
		}
	}

	return resolveOperationContext(ctx, runtime, repositories, envSpec, fromBranch, toBranch)
}

func (a *App) resolveOperationEnvironmentsWithPrompt(ctx context.Context, reader *bufio.Reader, runtime commandRuntime, repositories []scm.Repository, spec string) ([]inspectsvc.Environment, error) {
	environments, err := resolveOperationEnvironments(ctx, runtime, repositories, spec)
	if err == nil {
		return environments, nil
	}
	if reader == nil {
		return nil, err
	}
	var topologyErr *topologyResolutionError
	if !errors.As(err, &topologyErr) {
		return nil, err
	}
	if strings.TrimSpace(topologyErr.Inference.Summary) != "" {
		fmt.Fprintln(a.stderr, topologyErr.Inference.Summary)
	}
	envSpec, promptErr := a.promptForRequiredCommandValue(reader, "Environment mapping")
	if promptErr != nil {
		return nil, promptErr
	}
	return resolveOperationEnvironments(ctx, runtime, repositories, envSpec)
}

func canPromptForPromotionIntent(err error, fromBranch, toBranch string) bool {
	if strings.TrimSpace(fromBranch) != "" && strings.TrimSpace(toBranch) != "" {
		return false
	}
	var topologyErr *topologyResolutionError
	if errors.As(err, &topologyErr) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "both --from and --to branches are required") ||
		strings.Contains(message, "unable to infer source branch")
}
