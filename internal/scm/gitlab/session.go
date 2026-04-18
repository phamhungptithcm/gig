package gitlab

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

func (s *Session) Status(ctx context.Context) error {
	return s.status(ctx)
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
	cmd.Stdin = commandStdin(s.stdin)
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("glab auth login failed: %w", err)
	}
	return nil
}

func commandStdin(input io.Reader) io.Reader {
	if file, ok := input.(*os.File); ok {
		return file
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

func (s *Session) ListRepositories(ctx context.Context, namespace string) ([]scm.Repository, error) {
	type projectPayload struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Archived          bool   `json:"archived"`
	}

	namespace = strings.Trim(strings.TrimSpace(namespace), "/")
	repositories := make([]scm.Repository, 0, 32)
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("projects?membership=true&simple=true&order_by=last_activity_at&sort=desc&per_page=100&page=%d", page)
		output, err := s.runGLab(ctx, "api", endpoint)
		if err != nil {
			return nil, err
		}

		var payload []projectPayload
		if err := json.Unmarshal([]byte(output), &payload); err != nil {
			return nil, fmt.Errorf("parse gitlab repository discovery response: %w", err)
		}
		if len(payload) == 0 {
			break
		}

		for _, project := range payload {
			if project.Archived {
				continue
			}
			pathWithNamespace := strings.Trim(project.PathWithNamespace, "/")
			if pathWithNamespace == "" {
				continue
			}
			if namespace != "" && !strings.HasPrefix(strings.ToLower(pathWithNamespace), strings.ToLower(namespace)+"/") && !strings.EqualFold(pathWithNamespace, namespace) {
				continue
			}
			repositories = append(repositories, scm.Repository{
				Name: strings.TrimSpace(project.Name),
				Root: "gitlab:" + pathWithNamespace,
				Type: scm.TypeGitLab,
			})
		}
	}

	return repositories, nil
}
