package releasenotes

import (
	"strings"
	"testing"
)

func TestGenerateMarkdownGroupsChangesByArea(t *testing.T) {
	got := GenerateMarkdown(Input{
		RepositoryName: "gig",
		PreviousTag:    "v2026.04.09",
		CurrentTag:     "v2026.04.13",
		CompareURL:     "https://github.com/phamhungptithcm/gig/compare/v2026.04.09...v2026.04.13",
		Subjects: []string{
			"feat(sourcecontrol): add provider-first remote audit base",
			"feat(cli): add manifest packets and workspace doctor",
			"fix(release): harden install and release flow",
			"docs(product): reset remote-first direction",
			"chore(repo): bootstrap github automation",
		},
	})

	for _, want := range []string{
		"## Summary",
		"This `v2026.04.13` release focuses on CLI and release workflows, source-control-native access, and packaging and release automation.",
		"- Changes captured: `5`",
		"- Product updates: `2`",
		"- Fixes: `1`",
		"- Docs updates: `1`",
		"- Maintenance updates: `1`",
		"### CLI and release workflows",
		"- Add manifest packets and workspace doctor",
		"### Source-control-native access",
		"- Add provider-first remote audit base",
		"### Packaging and release automation",
		"- Harden install and release flow",
		"### Docs and guidance",
		"- Reset remote-first direction",
		"### Engineering and maintenance",
		"- Bootstrap github automation",
		"## Full Changelog",
		"- Compare: [v2026.04.09...v2026.04.13](https://github.com/phamhungptithcm/gig/compare/v2026.04.09...v2026.04.13)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q\n%s", want, got)
		}
	}
}

func TestGenerateMarkdownIncludesUpgradeNotesForBreakingChanges(t *testing.T) {
	got := GenerateMarkdown(Input{
		PreviousTag: "v2026.04.09",
		CurrentTag:  "v2026.04.13",
		Subjects: []string{
			"feat(cli)!: remove legacy local-only defaults",
		},
	})

	for _, want := range []string{
		"## Upgrade Notes",
		"- Remove legacy local-only defaults",
		"- Breaking changes: `1`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q\n%s", want, got)
		}
	}
}

func TestGenerateMarkdownFallsBackToInitialReleaseTemplate(t *testing.T) {
	got := GenerateMarkdown(Input{
		RepositoryName: "gig",
		CurrentTag:     "v2026.04.13",
	})

	for _, want := range []string{
		"## Summary",
		"`gig` ships as a remote-first release audit CLI",
		"## What ships in this release",
		"- Promotion planning and Markdown or JSON release packets",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q\n%s", want, got)
		}
	}
}
