package azuredevops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"gig/internal/scm"
)

const azureDevOpsResourceID = "499b84ac-1321-427f-aa17-267ca6975798"

type commandRunner func(ctx context.Context, args ...string) (string, error)

type Session struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	run    commandRunner
	client *http.Client
}

func NewSession(stdin io.Reader, stdout, stderr io.Writer) *Session {
	return &Session{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		client: http.DefaultClient,
	}
}

func (s *Session) EnsureAuthenticated(ctx context.Context) error {
	if err := s.status(ctx); err == nil {
		return nil
	}

	if s.stderr != nil {
		fmt.Fprintln(s.stderr, "Azure DevOps authentication is required. Starting az login...")
	}
	if err := s.login(ctx); err != nil {
		return err
	}
	if err := s.status(ctx); err != nil {
		return fmt.Errorf("azure authentication is still unavailable after login: %w", err)
	}

	return nil
}

func (s *Session) Login(ctx context.Context) error {
	return s.login(ctx)
}

func (s *Session) AccessToken(ctx context.Context) (string, error) {
	output, err := s.runAzure(ctx, "account", "get-access-token", "--resource", azureDevOpsResourceID, "--query", "accessToken", "--output", "tsv")
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(output)
	if token == "" {
		return "", fmt.Errorf("azure access token was empty")
	}

	return token, nil
}

func (s *Session) status(ctx context.Context) error {
	_, err := s.runAzure(ctx, "account", "show")
	return err
}

func (s *Session) login(ctx context.Context) error {
	if s.run != nil {
		_, err := s.run(ctx, "login")
		return err
	}
	if _, err := exec.LookPath("az"); err != nil {
		return fmt.Errorf("az executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "az", "login")
	cmd.Stdin = commandStdin(s.stdin)
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("az login failed: %w", err)
	}
	return nil
}

func commandStdin(input io.Reader) io.Reader {
	if file, ok := input.(*os.File); ok {
		return file
	}
	return nil
}

func (s *Session) runAzure(ctx context.Context, args ...string) (string, error) {
	if s.run != nil {
		return s.run(ctx, args...)
	}
	if _, err := exec.LookPath("az"); err != nil {
		return "", fmt.Errorf("az executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "az", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("az %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}

func (s *Session) ListRepositories(ctx context.Context, organization, project string) ([]scm.Repository, error) {
	type projectsPayload struct {
		Value []struct {
			Name string `json:"name"`
		} `json:"value"`
	}
	type repositoriesPayload struct {
		Value []struct {
			Name string `json:"name"`
		} `json:"value"`
	}

	organization = strings.TrimSpace(organization)
	project = strings.TrimSpace(project)
	if organization == "" {
		return nil, fmt.Errorf("azure devops repository discovery requires an organization")
	}

	token, err := s.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	projectsEndpoint := fmt.Sprintf("%s/%s/_apis/projects?$top=100&api-version=%s",
		strings.TrimRight(defaultBaseURL(), "/"),
		url.PathEscape(organization),
		apiVersion,
	)
	var projects projectsPayload
	if err := s.api(ctx, token, projectsEndpoint, &projects); err != nil {
		return nil, err
	}

	repositories := make([]scm.Repository, 0, 32)
	for _, discoveredProject := range projects.Value {
		projectName := strings.TrimSpace(discoveredProject.Name)
		if projectName == "" {
			continue
		}
		if project != "" && !strings.EqualFold(projectName, project) {
			continue
		}

		repositoriesEndpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories?api-version=%s",
			strings.TrimRight(defaultBaseURL(), "/"),
			url.PathEscape(organization),
			url.PathEscape(projectName),
			apiVersion,
		)
		var payload repositoriesPayload
		if err := s.api(ctx, token, repositoriesEndpoint, &payload); err != nil {
			return nil, err
		}
		for _, repository := range payload.Value {
			repositoryName := strings.TrimSpace(repository.Name)
			if repositoryName == "" {
				continue
			}
			repositories = append(repositories, scm.Repository{
				Name: repositoryName,
				Root: fmt.Sprintf("azure-devops:%s/%s/%s", organization, projectName, repositoryName),
				Type: scm.TypeAzureDevOps,
			})
		}
	}

	return repositories, nil
}

func (s *Session) api(ctx context.Context, token, endpoint string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")

	client := s.client
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("azure devops repository discovery failed: %s", message)
	}
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		return fmt.Errorf("parse azure devops repository discovery response: %w", err)
	}
	return nil
}
