package svn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type credentialStore interface {
	Load() (credentials, error)
	Save(credentials) error
}

type fileCredentialStore struct {
	path string
}

func (s fileCredentialStore) Load() (credentials, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		return credentials{}, err
	}

	var creds credentials
	if err := json.Unmarshal(content, &creds); err != nil {
		return credentials{}, fmt.Errorf("parse svn credentials: %w", err)
	}
	if strings.TrimSpace(creds.Username) == "" || strings.TrimSpace(creds.Password) == "" {
		return credentials{}, fmt.Errorf("svn credentials file is incomplete")
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

	return os.WriteFile(s.path, content, 0o600)
}

func authFilePath() string {
	if value := strings.TrimSpace(os.Getenv("GIG_SVN_AUTH_FILE")); value != "" {
		return value
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ".gig-svn.json"
		}
		return filepath.Join(homeDir, ".gig", "svn.json")
	}

	return filepath.Join(configDir, "gig", "svn.json")
}
