package svn

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Session struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	run    commandRunner
	store  credentialStore
}

func NewSession(stdin io.Reader, stdout, stderr io.Writer) *Session {
	return &Session{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		store:  fileCredentialStore{path: authFilePath()},
	}
}

func (s *Session) EnsureAuthenticated(ctx context.Context, repositoryRoots ...string) error {
	if err := s.ensureExecutable(); err != nil {
		return err
	}

	creds, err := s.Credentials(ctx)
	if err == nil {
		if err := s.validateAgainstRepository(ctx, creds, repositoryRoots...); err == nil {
			return nil
		}
	}

	if s.stderr != nil {
		fmt.Fprintln(s.stderr, "SVN credentials are required. Starting interactive SVN login...")
	}
	if err := s.Login(ctx); err != nil {
		return err
	}

	creds, err = s.Credentials(ctx)
	if err != nil {
		return err
	}
	if err := s.validateAgainstRepository(ctx, creds, repositoryRoots...); err != nil {
		return fmt.Errorf("svn authentication is still unavailable after login: %w", err)
	}

	return nil
}

func (s *Session) Login(ctx context.Context) error {
	if err := s.ensureExecutable(); err != nil {
		return err
	}

	if username, password, ok := envCredentials(); ok {
		creds := credentials{Username: username, Password: password}
		if err := s.validateAgainstRepository(ctx, creds); err != nil {
			return fmt.Errorf("svn credentials from environment are invalid: %w", err)
		}
		return nil
	}

	if creds, err := s.credentialStore().Load(); err == nil {
		if err := s.validateAgainstRepository(ctx, creds); err == nil {
			return nil
		}
	}

	creds, repositoryURL, err := s.promptCredentials()
	if err != nil {
		return err
	}
	if err := s.validateAgainstRepository(ctx, creds, repositoryURL); err != nil {
		return fmt.Errorf("svn credentials were rejected: %w", err)
	}
	return s.credentialStore().Save(creds)
}

func (s *Session) Credentials(context.Context) (credentials, error) {
	if username, password, ok := envCredentials(); ok {
		return credentials{Username: username, Password: password}, nil
	}
	return s.credentialStore().Load()
}

func (s *Session) promptCredentials() (credentials, string, error) {
	input := s.stdin
	if input == nil {
		input = os.Stdin
	}
	reader := bufio.NewReader(input)

	prompt := s.stderr
	if prompt == nil {
		prompt = s.stdout
	}

	repositoryURL := ""
	if prompt != nil {
		fmt.Fprint(prompt, "SVN repository URL (optional, press Enter to skip validation): ")
	}
	repositoryURL, _ = reader.ReadString('\n')
	repositoryURL = strings.TrimSpace(repositoryURL)

	if prompt != nil {
		fmt.Fprint(prompt, "SVN username: ")
	}
	username, err := reader.ReadString('\n')
	if err != nil && len(strings.TrimSpace(username)) == 0 {
		return credentials{}, "", fmt.Errorf("svn username is required")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return credentials{}, "", fmt.Errorf("svn username is required")
	}

	if prompt != nil {
		fmt.Fprint(prompt, "SVN password: ")
	}
	password, err := readSecret(input, reader)
	if err != nil {
		return credentials{}, "", err
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return credentials{}, "", fmt.Errorf("svn password is required")
	}
	if prompt != nil {
		fmt.Fprintln(prompt)
	}

	return credentials{
		Username: username,
		Password: password,
	}, repositoryURL, nil
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
		return "", fmt.Errorf("svn password is required")
	}
	return value, nil
}

func (s *Session) validateAgainstRepository(ctx context.Context, creds credentials, repositoryRoots ...string) error {
	if len(repositoryRoots) == 0 {
		return nil
	}

	for _, repositoryRoot := range repositoryRoots {
		repositoryRoot = strings.TrimSpace(repositoryRoot)
		if repositoryRoot == "" {
			continue
		}

		target := repositoryRoot
		if strings.HasPrefix(strings.ToLower(target), "svn:") {
			target = strings.TrimSpace(target[len("svn:"):])
		}
		if target == "" {
			continue
		}

		if _, err := s.runSVN(ctx,
			"--non-interactive",
			"--username", creds.Username,
			"--password", creds.Password,
			"info",
			"--xml",
			target,
		); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (s *Session) ensureExecutable() error {
	if s.run != nil {
		return nil
	}
	if _, err := exec.LookPath("svn"); err != nil {
		return fmt.Errorf("svn executable not found: %w", err)
	}
	return nil
}

func (s *Session) runSVN(ctx context.Context, args ...string) (string, error) {
	if s.run != nil {
		return s.run(ctx, args...)
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

func envCredentials() (string, string, bool) {
	username := strings.TrimSpace(os.Getenv("GIG_SVN_USERNAME"))
	password := strings.TrimSpace(os.Getenv("GIG_SVN_PASSWORD"))
	if username == "" || password == "" {
		return "", "", false
	}
	return username, password, true
}

func (s *Session) credentialStore() credentialStore {
	if s.store == nil {
		s.store = fileCredentialStore{path: authFilePath()}
	}
	return s.store
}
