package github

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Session struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	run    commandRunner
}

func NewSession(stdin io.Reader, stdout, stderr io.Writer) *Session {
	return &Session{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

func (s *Session) EnsureAuthenticated(ctx context.Context) error {
	if err := s.status(ctx); err == nil {
		return nil
	}

	if s.stderr != nil {
		fmt.Fprintln(s.stderr, "GitHub authentication is required. Starting gh auth login...")
	}
	if err := s.login(ctx); err != nil {
		return err
	}
	if err := s.status(ctx); err != nil {
		return fmt.Errorf("github authentication is still unavailable after login: %w", err)
	}

	return nil
}

func (s *Session) Login(ctx context.Context) error {
	return s.login(ctx)
}

func (s *Session) status(ctx context.Context) error {
	_, err := s.runGH(ctx, "auth", "status", "--hostname", "github.com")
	return err
}

func (s *Session) login(ctx context.Context) error {
	if s.run != nil {
		_, err := s.run(ctx, "auth", "login", "--hostname", "github.com", "--web")
		return err
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "gh", "auth", "login", "--hostname", "github.com", "--web")
	cmd.Stdin = s.stdin
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh auth login failed: %w", err)
	}
	return nil
}

func (s *Session) runGH(ctx context.Context, args ...string) (string, error) {
	if s.run != nil {
		return s.run(ctx, args...)
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}
