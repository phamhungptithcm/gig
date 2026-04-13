package bitbucket

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var errCredentialsNotFound = errors.New("bitbucket credentials not found")

const (
	bitbucketKeychainService = "com.hunpeolabs.gig.bitbucket"
	bitbucketKeychainAccount = "bitbucket-cloud"
)

type credentialStore interface {
	Load() (credentials, error)
	Save(credentials) error
}

type compositeCredentialStore struct {
	stores []credentialStore
}

type fileCredentialStore struct {
	path string
}

type securityRunner func(args ...string) (string, error)

type keychainCredentialStore struct {
	run securityRunner
}

func newDefaultCredentialStore() credentialStore {
	if strings.TrimSpace(os.Getenv("GIG_BITBUCKET_AUTH_FILE")) != "" {
		return fileCredentialStore{path: authFilePath()}
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GIG_BITBUCKET_DISABLE_KEYCHAIN")), "1") {
		return fileCredentialStore{path: authFilePath()}
	}
	if runtime.GOOS == "darwin" {
		return compositeCredentialStore{
			stores: []credentialStore{
				keychainCredentialStore{},
				fileCredentialStore{path: authFilePath()},
			},
		}
	}
	return fileCredentialStore{path: authFilePath()}
}

func (s compositeCredentialStore) Load() (credentials, error) {
	var lastErr error
	for _, store := range s.stores {
		creds, err := store.Load()
		if err == nil {
			return creds, nil
		}
		if errors.Is(err, errCredentialsNotFound) {
			lastErr = err
			continue
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errCredentialsNotFound
	}
	return credentials{}, lastErr
}

func (s compositeCredentialStore) Save(creds credentials) error {
	var lastErr error
	for _, store := range s.stores {
		if err := store.Save(creds); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no credential store is configured")
	}
	return lastErr
}

func (s fileCredentialStore) Load() (credentials, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return credentials{}, errCredentialsNotFound
		}
		return credentials{}, err
	}

	var creds credentials
	if err := json.Unmarshal(content, &creds); err != nil {
		return credentials{}, fmt.Errorf("parse bitbucket credentials: %w", err)
	}
	if strings.TrimSpace(creds.Email) == "" || strings.TrimSpace(creds.APIToken) == "" {
		return credentials{}, fmt.Errorf("bitbucket credentials file is incomplete")
	}
	return creds, nil
}

func (s fileCredentialStore) Save(creds credentials) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	content, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	if err := os.WriteFile(s.path, content, 0o600); err != nil {
		return err
	}

	return nil
}

func (s keychainCredentialStore) Load() (credentials, error) {
	output, err := s.runSecurity("find-generic-password", "-a", bitbucketKeychainAccount, "-s", bitbucketKeychainService, "-w")
	if err != nil {
		if isKeychainItemMissing(err) {
			return credentials{}, errCredentialsNotFound
		}
		return credentials{}, fmt.Errorf("load bitbucket credentials from macOS Keychain: %w", err)
	}

	var creds credentials
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &creds); err != nil {
		return credentials{}, fmt.Errorf("parse bitbucket credentials from macOS Keychain: %w", err)
	}
	if strings.TrimSpace(creds.Email) == "" || strings.TrimSpace(creds.APIToken) == "" {
		return credentials{}, fmt.Errorf("bitbucket credentials in macOS Keychain are incomplete")
	}
	return creds, nil
}

func (s keychainCredentialStore) Save(creds credentials) error {
	payload, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	if _, err := s.runSecurity(
		"add-generic-password",
		"-a", bitbucketKeychainAccount,
		"-s", bitbucketKeychainService,
		"-U",
		"-w", string(payload),
	); err != nil {
		return fmt.Errorf("save bitbucket credentials to macOS Keychain: %w", err)
	}

	return nil
}

func (s keychainCredentialStore) runSecurity(args ...string) (string, error) {
	if s.run != nil {
		return s.run(args...)
	}
	if _, err := exec.LookPath("security"); err != nil {
		return "", err
	}

	cmd := exec.Command("security", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("security %s failed: %s", strings.Join(args, " "), message)
	}
	return string(output), nil
}

func isKeychainItemMissing(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "could not be found in the keychain")
}

func authFilePath() string {
	if value := strings.TrimSpace(os.Getenv("GIG_BITBUCKET_AUTH_FILE")); value != "" {
		return value
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ".gig-bitbucket.json"
		}
		return filepath.Join(homeDir, ".gig", "bitbucket.json")
	}

	return filepath.Join(configDir, "gig", "bitbucket.json")
}
