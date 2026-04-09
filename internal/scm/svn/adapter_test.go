package svn

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gig/internal/config"
	"gig/internal/scm"
	"gig/internal/ticket"
)

func TestBuildCommitsFiltersTicketAndInfersBranches(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	commits := buildCommits([]logEntry{
		{
			Revision: "123",
			Message:  "ABC-123 fix Mendix deploy flow\nMore detail",
			Paths: []logPath{
				{Value: "/branches/dev/mendix/HorizonCRM.mpr"},
				{Value: "/branches/dev/javasource/App.java"},
			},
		},
		{
			Revision: "124",
			Message:  "XYZ-999 unrelated ticket",
			Paths: []logPath{
				{Value: "/trunk/README.md"},
			},
		},
	}, parser, "ABC-123", "")

	if len(commits) != 1 {
		t.Fatalf("buildCommits() returned %d commits, want 1", len(commits))
	}
	if commits[0].Hash != "r123" {
		t.Fatalf("buildCommits() hash = %q, want %q", commits[0].Hash, "r123")
	}
	if commits[0].Subject != "ABC-123 fix Mendix deploy flow" {
		t.Fatalf("buildCommits() subject = %q", commits[0].Subject)
	}
	if len(commits[0].Branches) != 1 || commits[0].Branches[0] != "dev" {
		t.Fatalf("buildCommits() branches = %#v, want [dev]", commits[0].Branches)
	}
}

func TestBuildCommitsUsesRequestedBranchWhenProvided(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	commits := buildCommits([]logEntry{
		{
			Revision: "200",
			Message:  "ABC-123 promotion candidate",
			Paths: []logPath{
				{Value: "/branches/test/mendix/HorizonCRM.mpr"},
			},
		},
	}, parser, "ABC-123", "test")

	if len(commits) != 1 {
		t.Fatalf("buildCommits() returned %d commits, want 1", len(commits))
	}
	if got := commits[0].Branches; len(got) != 1 || got[0] != "test" {
		t.Fatalf("buildCommits() branches = %#v, want [test]", got)
	}
}

func TestChangedFilesNormalizesBranchPrefixes(t *testing.T) {
	t.Parallel()

	files := changedFiles([]logPath{
		{Value: "/branches/dev/db/migrations/001_add_column.sql"},
		{Value: "/branches/dev/mendix/HorizonCRM.mpr"},
		{Value: "/branches/dev/mendix/HorizonCRM.mpr"},
		{Value: "/trunk/README.md"},
	})

	want := []string{
		"README.md",
		"db/migrations/001_add_column.sql",
		"mendix/HorizonCRM.mpr",
	}
	if len(files) != len(want) {
		t.Fatalf("changedFiles() returned %d files, want %d (%#v)", len(files), len(want), files)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("changedFiles()[%d] = %q, want %q", i, files[i], want[i])
		}
	}
}

func TestResolveBranchPathUsesStandardSVNLayout(t *testing.T) {
	t.Parallel()

	gotPath, gotDisplay := resolveBranchPath("test", "^/trunk")
	if gotPath != "branches/test" || gotDisplay != "test" {
		t.Fatalf("resolveBranchPath(trunk) = (%q, %q), want (%q, %q)", gotPath, gotDisplay, "branches/test", "test")
	}

	gotPath, gotDisplay = resolveBranchPath("trunk", "^/branches/dev")
	if gotPath != "trunk" || gotDisplay != "trunk" {
		t.Fatalf("resolveBranchPath(branches/dev) = (%q, %q), want (%q, %q)", gotPath, gotDisplay, "trunk", "trunk")
	}
}

func TestResolveBranchPathPreservesNestedProjectSuffix(t *testing.T) {
	t.Parallel()

	gotPath, gotDisplay := resolveBranchPath("test", "^/branches/dev/HorizonCRM")
	if gotPath != "branches/test/HorizonCRM" || gotDisplay != "test" {
		t.Fatalf("resolveBranchPath(nested) = (%q, %q), want (%q, %q)", gotPath, gotDisplay, "branches/test/HorizonCRM", "test")
	}
}

func TestDisplayBranchNameHandlesCommonLayouts(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"^/trunk":            "trunk",
		"^/branches/dev":     "dev",
		"^/tags/release-1.0": "tags/release-1.0",
		"^/projects/horizon": "projects/horizon",
		"/branches/test/app": "test",
		"branches/main":      "main",
	}

	for input, want := range cases {
		if got := displayBranchName(input); got != want {
			t.Fatalf("displayBranchName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRevisionHash(t *testing.T) {
	t.Parallel()

	if got := revisionHash("123"); got != "r123" {
		t.Fatalf("revisionHash() = %q, want %q", got, "r123")
	}
	if got := revisionHash("r456"); got != "r456" {
		t.Fatalf("revisionHash() with prefix = %q, want %q", got, "r456")
	}
}

func TestAdapterType(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	if got := NewAdapter(parser).Type(); got != scm.TypeSVN {
		t.Fatalf("Type() = %q, want %q", got, scm.TypeSVN)
	}
}

func TestAdapterCompareBranchesUsesMergeinfoEligibility(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	infoXML := `<info><entry><url>https://svn.example.com/repos/app/branches/dev/HorizonCRM</url><relative-url>^/branches/dev/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`
	sourceLogXML := `<log>
<logentry revision="101"><msg>ABC-123 initial change</msg><paths><path>/branches/dev/HorizonCRM/javasource/Main.java</path></paths></logentry>
<logentry revision="102"><msg>ABC-123 follow-up fix</msg><paths><path>/branches/dev/HorizonCRM/db/migrations/001_add_column.sql</path></paths></logentry>
</log>`
	targetLogXML := `<log>
<logentry revision="220"><msg>ABC-123 test-only patch</msg><paths><path>/branches/test/HorizonCRM/javasource/Test.java</path></paths></logentry>
</log>`

	adapter := &Adapter{
		parser: parser,
		run: func(_ context.Context, args ...string) (string, error) {
			command := strings.Join(args, " ")
			switch command {
			case "info --xml /workspace/svnrepo":
				return infoXML, nil
			case "log --xml --verbose https://svn.example.com/repos/app/branches/dev/HorizonCRM":
				return sourceLogXML, nil
			case "log --xml --verbose https://svn.example.com/repos/app/branches/test/HorizonCRM":
				return targetLogXML, nil
			case "mergeinfo --show-revs eligible https://svn.example.com/repos/app/branches/dev/HorizonCRM https://svn.example.com/repos/app/branches/test/HorizonCRM":
				return "r102\n", nil
			default:
				return "", fmt.Errorf("unexpected svn call: %s", command)
			}
		},
	}

	result, err := adapter.CompareBranches(context.Background(), "/workspace/svnrepo", scm.CompareQuery{
		TicketID:   "ABC-123",
		FromBranch: "dev",
		ToBranch:   "test",
	})
	if err != nil {
		t.Fatalf("CompareBranches() error = %v", err)
	}

	if len(result.SourceCommits) != 2 {
		t.Fatalf("len(SourceCommits) = %d, want 2", len(result.SourceCommits))
	}
	if len(result.TargetCommits) != 1 {
		t.Fatalf("len(TargetCommits) = %d, want 1", len(result.TargetCommits))
	}
	if len(result.MissingCommits) != 1 {
		t.Fatalf("len(MissingCommits) = %d, want 1", len(result.MissingCommits))
	}
	if result.MissingCommits[0].Hash != "r102" {
		t.Fatalf("MissingCommits[0].Hash = %q, want %q", result.MissingCommits[0].Hash, "r102")
	}
}
