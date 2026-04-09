package dependency_test

import (
	"reflect"
	"testing"

	"gig/internal/config"
	"gig/internal/dependency"
	"gig/internal/ticket"
)

func TestParserParseCommitMessageExtractsDeclaredDependencies(t *testing.T) {
	t.Parallel()

	parser := newParser(t)
	message := "ABC-123 | accounts-api | wire dependency checks\n\nDepends-On: XYZ-456\n"

	got, err := parser.ParseCommitMessage("ABC-123", "abc12345", "ABC-123 | accounts-api | wire dependency checks", message)
	if err != nil {
		t.Fatalf("ParseCommitMessage() error = %v", err)
	}

	want := []dependency.DeclaredDependency{
		{
			TicketID:      "ABC-123",
			DependsOn:     "XYZ-456",
			CommitHash:    "abc12345",
			CommitSubject: "ABC-123 | accounts-api | wire dependency checks",
			TrailerKey:    dependency.TrailerDependsOn,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCommitMessage() = %#v, want %#v", got, want)
	}
}

func TestParserParseCommitMessageDeduplicatesAndNormalizesDependencies(t *testing.T) {
	t.Parallel()

	parser := newParser(t)
	message := "ABC-123 | accounts-api | wire dependency checks\n\nDepends-On: xyz-456\ndepends-on: OPS-99, xyz-456\nDepends-On: ABC-123\n"

	got, err := parser.ParseCommitMessage("abc-123", "abc12345", "ABC-123 | accounts-api | wire dependency checks", message)
	if err != nil {
		t.Fatalf("ParseCommitMessage() error = %v", err)
	}

	want := []dependency.DeclaredDependency{
		{
			TicketID:      "ABC-123",
			DependsOn:     "XYZ-456",
			CommitHash:    "abc12345",
			CommitSubject: "ABC-123 | accounts-api | wire dependency checks",
			TrailerKey:    dependency.TrailerDependsOn,
		},
		{
			TicketID:      "ABC-123",
			DependsOn:     "OPS-99",
			CommitHash:    "abc12345",
			CommitSubject: "ABC-123 | accounts-api | wire dependency checks",
			TrailerKey:    dependency.TrailerDependsOn,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCommitMessage() = %#v, want %#v", got, want)
	}
}

func TestParserParseCommitMessageIgnoresInvalidAndUnsupportedTrailers(t *testing.T) {
	t.Parallel()

	parser := newParser(t)
	message := "ABC-123 | accounts-api | wire dependency checks\n\nRisk: manual-review\nDepends-On: not-a-ticket, XYZ_7, UI-7\n"

	got, err := parser.ParseCommitMessage("ABC-123", "abc12345", "ABC-123 | accounts-api | wire dependency checks", message)
	if err != nil {
		t.Fatalf("ParseCommitMessage() error = %v", err)
	}

	want := []dependency.DeclaredDependency{
		{
			TicketID:      "ABC-123",
			DependsOn:     "UI-7",
			CommitHash:    "abc12345",
			CommitSubject: "ABC-123 | accounts-api | wire dependency checks",
			TrailerKey:    dependency.TrailerDependsOn,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCommitMessage() = %#v, want %#v", got, want)
	}
}

func TestParserParseCommitMessageReturnsErrorForInvalidPrimaryTicket(t *testing.T) {
	t.Parallel()

	parser := newParser(t)

	if _, err := parser.ParseCommitMessage("missing-ticket", "abc12345", "bad", "Depends-On: XYZ-456"); err == nil {
		t.Fatal("ParseCommitMessage() expected error for invalid ticket ID")
	}
}

func newParser(t *testing.T) dependency.Parser {
	t.Helper()

	tickets, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	return dependency.NewParser(tickets)
}
