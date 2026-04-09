package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	diffsvc "gig/internal/diff"
	"gig/internal/scm"
	ticketsvc "gig/internal/ticket"
)

func RenderScan(w io.Writer, basePath string, repositories []scm.Repository) error {
	if len(repositories) == 0 {
		_, err := fmt.Fprintf(w, "No repositories found under %s.\n", basePath)
		return err
	}

	if _, err := fmt.Fprintf(w, "Found %d repositories under %s.\n\n", len(repositories), basePath); err != nil {
		return err
	}

	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "REPO\tSCM\tBRANCH\tPATH"); err != nil {
		return err
	}

	for _, repository := range repositories {
		branch := repository.CurrentBranch
		if branch == "" {
			branch = "-"
		}

		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", repository.Name, repository.Type, branch, repository.Root); err != nil {
			return err
		}
	}

	return table.Flush()
}

func RenderFind(w io.Writer, ticketID, basePath string, scannedRepoCount int, results []ticketsvc.SearchResult) error {
	if _, err := fmt.Fprintf(w, "Ticket %s\n", ticketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", basePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n\n", scannedRepoCount); err != nil {
		return err
	}

	if scannedRepoCount == 0 {
		_, err := fmt.Fprintf(w, "No repositories found under %s.\n", basePath)
		return err
	}

	if len(results) == 0 {
		_, err := fmt.Fprintf(w, "No commits found for %s.\n", ticketID)
		return err
	}

	for _, result := range results {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", result.Repository.Name, result.Repository.Root, result.Repository.Type); err != nil {
			return err
		}

		for _, commit := range result.Commits {
			if _, err := fmt.Fprintf(w, "  - %s  %s\n", commit.ShortHash(), commit.Subject); err != nil {
				return err
			}
			if len(commit.Branches) > 0 {
				if _, err := fmt.Fprintf(w, "    branches: %s\n", strings.Join(commit.Branches, ", ")); err != nil {
					return err
				}
			}
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}

func RenderDiff(w io.Writer, ticketID, fromBranch, toBranch, basePath string, scannedRepoCount int, results []diffsvc.Result) error {
	if _, err := fmt.Fprintf(w, "Ticket %s\n", ticketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", basePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Compare: %s -> %s\n", fromBranch, toBranch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n\n", scannedRepoCount); err != nil {
		return err
	}

	if scannedRepoCount == 0 {
		_, err := fmt.Fprintf(w, "No repositories found under %s.\n", basePath)
		return err
	}

	if len(results) == 0 {
		_, err := fmt.Fprintln(w, "No matching commits found across detected repositories.")
		return err
	}

	for _, result := range results {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", result.Repository.Name, result.Repository.Root, result.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  source commits: %d\n", len(result.Compare.SourceCommits)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  target commits: %d\n", len(result.Compare.TargetCommits)); err != nil {
			return err
		}

		if len(result.Compare.MissingCommits) == 0 {
			if _, err := fmt.Fprintf(w, "  missing in %s: none\n\n", toBranch); err != nil {
				return err
			}
			continue
		}

		if _, err := fmt.Fprintf(w, "  missing in %s:\n", toBranch); err != nil {
			return err
		}
		for _, commit := range result.Compare.MissingCommits {
			if _, err := fmt.Fprintf(w, "    - %s  %s\n", commit.ShortHash(), commit.Subject); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}
