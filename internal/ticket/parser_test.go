package ticket_test

import (
	"reflect"
	"testing"

	"gig/internal/config"
	"gig/internal/ticket"
)

func TestParserFindAllNormalizesAndDeduplicates(t *testing.T) {
	t.Parallel()

	parser := newParser(t)
	got := parser.FindAll("ABC-123 fix login, abc-123 retry, and XYZ-7 add test")
	want := []string{"ABC-123", "XYZ-7"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAll() = %#v, want %#v", got, want)
	}
}

func TestParserValidate(t *testing.T) {
	t.Parallel()

	parser := newParser(t)

	if err := parser.Validate("ABC-123"); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
	if err := parser.Validate("missing-ticket"); err == nil {
		t.Fatal("Validate() expected error for invalid ticket ID")
	}
}

func TestParserMatches(t *testing.T) {
	t.Parallel()

	parser := newParser(t)

	if !parser.Matches("ABC-123", "abc-123 | service-a | adjust validation") {
		t.Fatal("Matches() = false, want true")
	}
	if parser.Matches("ABC-123", "XYZ-7 unrelated change") {
		t.Fatal("Matches() = true, want false")
	}
}

func newParser(t *testing.T) ticket.Parser {
	t.Helper()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	return parser
}
