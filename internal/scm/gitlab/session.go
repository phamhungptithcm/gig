package gitlab

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
		fmt.Fprintln(s.stderr, "GitLab authentication is required. Starting glab auth login...")
	}
	if err := s.login(ctx); err != nil {
		return err
	}
	if err := s.status(ctx); err != nil {
		return fmt.Errorf("gitlab authentication is still unavailable after login: %w", err)
	}

	return nil
}

func (s *Session) Login(ctx context.Context) error {
	return s.login(ctx)
}

func (s *Session) status(ctx context.Context) error {
	_, err := s.runGLab(ctx, "auth", "status", "--hostname", "gitlab.com")
	return err
}

func (s *Session) login(ctx context.Context) error {
	if s.run != nil {
		_, err := s.run(ctx, "auth", "login", "--hostname", "gitlab.com", "--web")
		return err
	}
	if _, err := exec.LookPath("glab"); err != nil {
		return fmt.Errorf("glab executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "glab", "auth", "login", "--hostname", "gitlab.com", "--web")
	cmd.Stdin = s.stdin
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("glab auth login failed: %w", err)
	}
	return nil
}

func (s *Session) runGLab(ctx context.Context, args ...string) (string, error) {
	if s.run != nil {
		return s.run(ctx, args...)
	}
	if _, err := exec.LookPath("glab"); err != nil {
		return "", fmt.Errorf("glab executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "glab", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("glab %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}
