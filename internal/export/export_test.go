package exporter

import (
	"bytes"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
)

func TestResolveOutputFormatInfersFromOutExtension(t *testing.T) {
	t.Parallel()

	got, err := ResolveOutputFormat(ResolveOptions{
		RawFormat:     "human",
		OutputPath:    "verify.xlsx",
		DefaultFormat: FormatHuman,
	})
	if err != nil {
		t.Fatalf("ResolveOutputFormat() error = %v", err)
	}
	if got.Format != FormatXLSX || got.Target != TargetFile {
		t.Fatalf("ResolveOutputFormat() = %#v, want xlsx file", got)
	}

	got, err = ResolveOutputFormat(ResolveOptions{
		RawFormat:     "markdown",
		OutputPath:    "release-packet/",
		DefaultFormat: FormatMarkdown,
	})
	if err != nil {
		t.Fatalf("ResolveOutputFormat(directory) error = %v", err)
	}
	if got.Format != FormatCSV || got.Target != TargetDirectory {
		t.Fatalf("ResolveOutputFormat(directory) = %#v, want csv directory", got)
	}
}

func TestResolveOutputFormatReportsConflict(t *testing.T) {
	t.Parallel()

	_, err := ResolveOutputFormat(ResolveOptions{
		RawFormat:      "csv",
		FormatExplicit: true,
		OutputPath:     "verify.xlsx",
		DefaultFormat:  FormatHuman,
	})
	var conflict *FormatConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("ResolveOutputFormat() error = %T %[1]v, want FormatConflictError", err)
	}
	if conflict.Requested != FormatCSV || conflict.Inferred != FormatXLSX {
		t.Fatalf("conflict = %#v, want csv vs xlsx", conflict)
	}
}

func TestWriteSingleCSVHeadersEscapingAndDeterministicRows(t *testing.T) {
	t.Parallel()

	plan := samplePromotionPlan()
	verification := plansvc.BuildVerification(plan)
	releaseExport := BuildVerificationExport([]plansvc.PromotionPlan{plan}, []plansvc.Verification{verification}, sampleOptions())

	var buffer bytes.Buffer
	if err := WriteSingleCSV(&buffer, releaseExport.SingleCSV); err != nil {
		t.Fatalf("WriteSingleCSV() error = %v", err)
	}
	if strings.Contains(buffer.String(), "\x1b[") {
		t.Fatalf("CSV contains ANSI escape sequence: %q", buffer.String())
	}

	records, err := csv.NewReader(strings.NewReader(buffer.String())).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	wantHeader := []string{"Ticket", "Repository", "Check", "Result", "Details", "Evidence", "Next action"}
	if !reflect.DeepEqual(records[0], wantHeader) {
		t.Fatalf("header = %#v, want %#v", records[0], wantHeader)
	}
	if len(records) < 2 || records[1][0] != "ABC-123" || records[1][1] != "payments" {
		t.Fatalf("records = %#v, want deterministic first verification row", records)
	}

	var commits bytes.Buffer
	commitsSheet := releaseExport.Sheets[4]
	if err := WriteSingleCSV(&commits, commitsSheet); err != nil {
		t.Fatalf("WriteSingleCSV(commits) error = %v", err)
	}
	if !strings.Contains(commits.String(), "'=ABC-123 formula subject") {
		t.Fatalf("CSV = %q, want formula injection escape", commits.String())
	}
}

func TestWriteCSVDirectoryIncludesEmptyFilesWithHeaders(t *testing.T) {
	t.Parallel()

	releaseExport := BuildReleasePacketExport(nil, sampleOptions())
	dir := t.TempDir()
	if err := WriteCSVDirectory(dir, releaseExport); err != nil {
		t.Fatalf("WriteCSVDirectory() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "risks.csv"))
	if err != nil {
		t.Fatalf("ReadFile(risks.csv) error = %v", err)
	}
	if !strings.HasPrefix(string(content), "Severity,Category,Finding,Impact,Evidence,Recommended action,Owner,Status\n") {
		t.Fatalf("risks.csv = %q, want header-only file", string(content))
	}
}

func TestWriteXLSXSheetNamesAndHeaders(t *testing.T) {
	t.Parallel()

	plan := samplePromotionPlan()
	verification := plansvc.BuildVerification(plan)
	releaseExport := BuildVerificationExport([]plansvc.PromotionPlan{plan}, []plansvc.Verification{verification}, sampleOptions())

	var buffer bytes.Buffer
	if err := WriteXLSX(&buffer, releaseExport); err != nil {
		t.Fatalf("WriteXLSX() error = %v", err)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer workbook.Close()

	wantSheets := []string{"Summary", "Decision", "Risks", "Missing Changes", "Commits", "Manual Steps", "Evidence", "Metadata"}
	if got := workbook.GetSheetList(); !reflect.DeepEqual(got, wantSheets) {
		t.Fatalf("sheets = %#v, want %#v", got, wantSheets)
	}
	rows, err := workbook.GetRows("Missing Changes")
	if err != nil {
		t.Fatalf("GetRows(Missing Changes) error = %v", err)
	}
	if !reflect.DeepEqual(rows[0], missingHeaders) {
		t.Fatalf("missing headers = %#v, want %#v", rows[0], missingHeaders)
	}
	if rows[1][3] != "'=ABC-123 formula subject" {
		t.Fatalf("formula-safe XLSX subject = %q", rows[1][3])
	}
}

func TestBuildReleasePacketExportSheetNames(t *testing.T) {
	t.Parallel()

	releaseExport := BuildReleasePacketExport(nil, sampleOptions())
	got := make([]string, 0, len(releaseExport.Sheets))
	for _, sheet := range releaseExport.Sheets {
		got = append(got, sheet.Name)
	}
	want := []string{"Cover", "Release Decision", "Scope", "Risks", "Missing Changes", "Commits", "Manual Steps", "Verification", "Approvals", "Evidence", "Metadata"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packet sheets = %#v, want %#v", got, want)
	}
}

func samplePromotionPlan() plansvc.PromotionPlan {
	return plansvc.PromotionPlan{
		TicketID:   "ABC-123",
		FromBranch: "staging",
		ToBranch:   "main",
		Summary: plansvc.Summary{
			ScannedRepositories:   1,
			TouchedRepositories:   1,
			WarningRepositories:   1,
			TotalCommitsToPromote: 1,
			TotalManualSteps:      1,
		},
		Verdict: plansvc.VerdictWarning,
		Repositories: []plansvc.RepositoryPlan{
			{
				Repository: scm.Repository{Name: "payments", Root: "github:acme/payments", Type: scm.TypeGitHub},
				Compare: scm.CompareResult{
					FromBranch: "staging",
					ToBranch:   "main",
					SourceCommits: []scm.Commit{{
						Hash:    "abcdef1234567890",
						Subject: "=ABC-123 formula subject",
					}},
					MissingCommits: []scm.Commit{{
						Hash:    "abcdef1234567890",
						Subject: "=ABC-123 formula subject",
					}},
				},
				RiskSignals: []inspectsvc.RiskSignal{{
					Code:     "db-change",
					Level:    "manual-review",
					Summary:  "Database migration changed.",
					Examples: []string{"db/migrations/001.sql"},
				}},
				ManualSteps: []plansvc.Action{{
					Code:    "review-db-rollout",
					Summary: "Review DB migration ordering before promotion.",
				}},
				ProviderEvidence: &scm.ProviderEvidence{
					Checks: []scm.CheckEvidence{{
						Context:    "ci/test",
						State:      "success",
						CommitHash: "abcdef1234567890",
					}},
				},
				Verdict: plansvc.VerdictWarning,
			},
		},
	}
}

func sampleOptions() Options {
	return Options{
		GeneratedAt:      time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
		Command:          "gig verify ABC-123 --out verify.xlsx",
		ScopeLabel:       "github:acme/payments",
		Mode:             "remote",
		Provider:         "github",
		WorkingDirectory: "github:acme/payments",
	}
}
