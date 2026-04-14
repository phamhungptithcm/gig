package bitbucket

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"gig/internal/scm"

	"golang.org/x/term"
)

const defaultAPIBaseURL = "https://api.bitbucket.org/2.0"

type credentials struct {
	Email    string `json:"email"`
	APIToken string `json:"apiToken"`
}

type Session struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	client *http.Client
	store  credentialStore
}

func NewSession(stdin io.Reader, stdout, stderr io.Writer) *Session {
	return &Session{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		client: http.DefaultClient,
		store:  newDefaultCredentialStore(),
	}
}

func (s *Session) EnsureAuthenticated(ctx context.Context) error {
	if email, token, ok := envCredentials(); ok {
		if err := s.status(ctx, credentials{Email: email, APIToken: token}); err != nil {
			return fmt.Errorf("bitbucket credentials from environment are invalid: %w", err)
		}
		return nil
	}

	creds, err := s.credentialStore().Load()
	if err == nil {
		if err := s.status(ctx, creds); err == nil {
			return nil
		}
	}

	if s.stderr != nil {
		fmt.Fprintln(s.stderr, "Bitbucket authentication is required. Starting interactive API token login...")
	}
	if err := s.Login(ctx); err != nil {
		return err
	}

	creds, err = s.loadCredentials()
	if err != nil {
		return err
	}
	if err := s.status(ctx, creds); err != nil {
		return fmt.Errorf("bitbucket authentication is still unavailable after login: %w", err)
	}

	return nil
}

func (s *Session) Login(ctx context.Context) error {
	if email, token, ok := envCredentials(); ok {
		if err := s.status(ctx, credentials{Email: email, APIToken: token}); err != nil {
			return fmt.Errorf("bitbucket credentials from environment are invalid: %w", err)
		}
		return nil
	}

	if creds, err := s.credentialStore().Load(); err == nil {
		if err := s.status(ctx, creds); err == nil {
			return nil
		}
	}

	creds, err := s.promptCredentials()
	if err != nil {
		return err
	}
	if err := s.status(ctx, creds); err != nil {
		return fmt.Errorf("bitbucket credentials were rejected: %w", err)
	}
	if err := s.credentialStore().Save(creds); err != nil {
		return err
	}
	return nil
}

func (s *Session) Status(ctx context.Context) error {
	if email, token, ok := envCredentials(); ok {
		return s.status(ctx, credentials{Email: email, APIToken: token})
	}

	creds, err := s.credentialStore().Load()
	if err != nil {
		return err
	}
	return s.status(ctx, creds)
}

func (s *Session) Credentials(context.Context) (credentials, error) {
	return s.loadCredentials()
}

func (s *Session) loadCredentials() (credentials, error) {
	if email, token, ok := envCredentials(); ok {
		return credentials{Email: email, APIToken: token}, nil
	}
	return s.credentialStore().Load()
}

func (s *Session) status(ctx context.Context, creds credentials) error {
	endpoint := strings.TrimRight(resolveAPIBaseURL(), "/") + "/repositories?pagelen=1"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(creds.Email, creds.APIToken)
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
		return fmt.Errorf("bitbucket api rejected credentials: %s", message)
	}

	return nil
}

func (s *Session) promptCredentials() (credentials, error) {
	input := s.stdin
	if input == nil {
		input = os.Stdin
	}
	reader := bufio.NewReader(input)

	emailPrompt := s.stderr
	if emailPrompt == nil {
		emailPrompt = s.stdout
	}
	if emailPrompt != nil {
		fmt.Fprint(emailPrompt, "Bitbucket email: ")
	}
	email, err := reader.ReadString('\n')
	if err != nil && len(strings.TrimSpace(email)) == 0 {
		return credentials{}, fmt.Errorf("bitbucket email is required")
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return credentials{}, fmt.Errorf("bitbucket email is required")
	}

	tokenPrompt := s.stderr
	if tokenPrompt == nil {
		tokenPrompt = s.stdout
	}
	if tokenPrompt != nil {
		fmt.Fprint(tokenPrompt, "Bitbucket API token: ")
	}

	token, err := readSecret(input, reader)
	if err != nil {
		return credentials{}, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return credentials{}, fmt.Errorf("bitbucket API token is required")
	}

	if tokenPrompt != nil {
		fmt.Fprintln(tokenPrompt)
	}

	return credentials{
		Email:    email,
		APIToken: token,
	}, nil
}

func readSecret(input io.Reader, reader *bufio.Reader) (string, error) {
	file, ok := input.(*os.File)
	if ok && term.IsTerminal(int(file.Fd())) {
		value, err := term.ReadPassword(int(file.Fd()))
		if err != nil {
			return "", err
		}
		return string(value), nil
	}

	value, err := reader.ReadString('\n')
	if err != nil && len(strings.TrimSpace(value)) == 0 {
		return "", fmt.Errorf("bitbucket API token is required")
	}
	return value, nil
}

func resolveAPIBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("GIG_BITBUCKET_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultAPIBaseURL
}

func envCredentials() (string, string, bool) {
	email := strings.TrimSpace(os.Getenv("GIG_BITBUCKET_EMAIL"))
	token := strings.TrimSpace(os.Getenv("GIG_BITBUCKET_API_TOKEN"))
	if email == "" || token == "" {
		return "", "", false
	}
	return email, token, true
}

func (s *Session) credentialStore() credentialStore {
	if s.store == nil {
		s.store = newDefaultCredentialStore()
	}
	return s.store
}

func (s *Session) ListRepositories(ctx context.Context, workspace string) ([]scm.Repository, error) {
	type repositoryPayload struct {
		Name      string `json:"name"`
		FullName  string `json:"full_name"`
		Slug      string `json:"slug"`
		Workspace struct {
			Slug string `json:"slug"`
		} `json:"workspace"`
	}
	type repositoriesPayload struct {
		Values []repositoryPayload `json:"values"`
		Next   string              `json:"next"`
	}

	creds, err := s.loadCredentials()
	if err != nil {
		return nil, err
	}

	workspace = strings.TrimSpace(workspace)
	repositories := make([]scm.Repository, 0, 32)
	client := s.client
	if client == nil {
		client = http.DefaultClient
	}

	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("%s/repositories?role=member&sort=-updated_on&pagelen=100&page=%d", strings.TrimRight(resolveAPIBaseURL(), "/"), page)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		request.SetBasicAuth(creds.Email, creds.APIToken)
		request.Header.Set("Accept", "application/json")

		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		var payload repositoriesPayload
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			response.Body.Close()
			return nil, fmt.Errorf("parse bitbucket repository discovery response: %w", err)
		}
		response.Body.Close()
		if response.StatusCode >= 400 {
			return nil, fmt.Errorf("bitbucket repository discovery failed: %s", response.Status)
		}
		if len(payload.Values) == 0 {
			break
		}

		for _, repository := range payload.Values {
			workspaceSlug := strings.TrimSpace(repository.Workspace.Slug)
			repoSlug := strings.TrimSpace(repository.Slug)
			if workspaceSlug == "" || repoSlug == "" {
				parts := strings.Split(strings.TrimSpace(repository.FullName), "/")
				if len(parts) == 2 {
					workspaceSlug = strings.TrimSpace(parts[0])
					repoSlug = strings.TrimSpace(parts[1])
				}
			}
			if workspaceSlug == "" || repoSlug == "" {
				continue
			}
			if workspace != "" && !strings.EqualFold(workspaceSlug, workspace) {
				continue
			}
			repositories = append(repositories, scm.Repository{
				Name: strings.TrimSpace(repository.Name),
				Root: fmt.Sprintf("bitbucket:%s/%s", workspaceSlug, repoSlug),
				Type: scm.TypeBitbucket,
			})
		}
		if payload.Next == "" {
			break
		}
	}

	sort.SliceStable(repositories, func(i, j int) bool {
		return repositories[i].Root < repositories[j].Root
	})
	return repositories, nil
}
