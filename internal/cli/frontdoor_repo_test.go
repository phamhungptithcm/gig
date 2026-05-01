package cli

import (
	"bufio"
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"gig/internal/workarea"
)

func TestResolveFrontDoorPromptRepositoryFindsSavedShortName(t *testing.T) {
	store := workarea.NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	if _, err := store.Save(workarea.Definition{
		Name:       "payments",
		RepoTarget: "github:acme/payments",
	}, true); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	app, err := NewAppWithIO(strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewAppWithIO() error = %v", err)
	}

	repository, err := app.resolveFrontDoorPromptRepository(context.Background(), bufio.NewReader(strings.NewReader("")), store, "payments", "")
	if err != nil {
		t.Fatalf("resolveFrontDoorPromptRepository() error = %v", err)
	}
	if repository.Root != "github:acme/payments" {
		t.Fatalf("Root = %q, want github:acme/payments", repository.Root)
	}
}

func TestResolveFrontDoorPromptRepositoryPicksAmbiguousShortName(t *testing.T) {
	store := workarea.NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	for _, definition := range []workarea.Definition{
		{Name: "payments-github", RepoTarget: "github:acme/payments"},
		{Name: "payments-gitlab", RepoTarget: "gitlab:platform/payments"},
	} {
		if _, err := store.Save(definition, false); err != nil {
			t.Fatalf("Save(%s) error = %v", definition.Name, err)
		}
	}

	var stdout, stderr bytes.Buffer
	app, err := NewAppWithIO(strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewAppWithIO() error = %v", err)
	}

	repository, err := app.resolveFrontDoorPromptRepository(context.Background(), bufio.NewReader(strings.NewReader("2\n")), store, "payments", "")
	if err != nil {
		t.Fatalf("resolveFrontDoorPromptRepository() error = %v", err)
	}
	if repository.Root != "gitlab:platform/payments" {
		t.Fatalf("Root = %q, want gitlab:platform/payments", repository.Root)
	}
	if !strings.Contains(stdout.String(), "Found repositories matching \"payments\"") {
		t.Fatalf("stdout = %q, want ambiguous picker heading", stdout.String())
	}
}

func TestInferFrontDoorSaveName(t *testing.T) {
	if got := inferFrontDoorSaveName("svn:https://svn.example.com/repos/app/branches/staging/ProductName"); got != "ProductName" {
		t.Fatalf("inferFrontDoorSaveName() = %q, want ProductName", got)
	}
	if got := inferFrontDoorSaveName("github:acme/payments"); got != "payments" {
		t.Fatalf("inferFrontDoorSaveName() = %q, want payments", got)
	}
}
