package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"gig/internal/scm"
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

func (s *Session) Status(ctx context.Context) error {
	return s.status(ctx)
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
	cmd.Stdin = commandStdin(s.stdin)
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh auth login failed: %w", err)
	}
	return nil
}

func commandStdin(input io.Reader) io.Reader {
	if file, ok := input.(*os.File); ok {
		return file
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

func (s *Session) ListRepositories(ctx context.Context, owner string) ([]scm.Repository, error) {
	type repositoryOwner struct {
		Login string `json:"login"`
	}
	type repositoryPayload struct {
		Name     string          `json:"name"`
		FullName string          `json:"full_name"`
		Archived bool            `json:"archived"`
		Disabled bool            `json:"disabled"`
		Owner    repositoryOwner `json:"owner"`
	}

	owner = strings.TrimSpace(owner)
	repositories := make([]scm.Repository, 0, 32)
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("user/repos?sort=updated&per_page=100&page=%d", page)
		output, err := s.runGH(ctx, "api", endpoint)
		if err != nil {
			return nil, err
		}

		var payload []repositoryPayload
		if err := json.Unmarshal([]byte(output), &payload); err != nil {
			return nil, fmt.Errorf("parse github repository discovery response: %w", err)
		}
		if len(payload) == 0 {
			break
		}

		for _, repository := range payload {
			if repository.Archived || repository.Disabled {
				continue
			}
			fullName := strings.TrimSpace(repository.FullName)
			if fullName == "" {
				continue
			}
			if owner != "" && !strings.EqualFold(strings.TrimSpace(repository.Owner.Login), owner) {
				continue
			}
			repositories = append(repositories, scm.Repository{
				Name: strings.TrimSpace(repository.Name),
				Root: "github:" + fullName,
				Type: scm.TypeGitHub,
			})
		}
	}

	return repositories, nil
}
