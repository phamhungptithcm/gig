package releasenotes

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type Input struct {
	RepositoryName string
	PreviousTag    string
	CurrentTag     string
	CompareURL     string
	Subjects       []string
}

type sectionKey string

const (
	sectionCLI          sectionKey = "cli"
	sectionSCM          sectionKey = "scm"
	sectionDistribution sectionKey = "distribution"
	sectionDocs         sectionKey = "docs"
	sectionMaintenance  sectionKey = "maintenance"
)

var (
	conventionalCommitPattern = regexp.MustCompile(`(?i)^([a-z]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$`)
	dateTagPattern            = regexp.MustCompile(`^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}$`)
)

type parsedCommit struct {
	Type        string
	Description string
	Breaking    bool
	Section     sectionKey
}

type summaryStats struct {
	Total       int
	Product     int
	Fixes       int
	Docs        int
	Maintenance int
	Breaking    int
}

func GenerateMarkdown(input Input) string {
	repositoryName := strings.TrimSpace(input.RepositoryName)
	if repositoryName == "" {
		repositoryName = "gig"
	}

	if shouldRenderInitialRelease(input.PreviousTag) {
		return renderInitialRelease(repositoryName)
	}

	commits := parseCommits(input.Subjects)
	return renderMarkdown(input, commits)
}

func shouldRenderInitialRelease(previousTag string) bool {
	previousTag = strings.TrimSpace(previousTag)
	return previousTag == "" || !dateTagPattern.MatchString(previousTag)
}

func parseCommits(subjects []string) []parsedCommit {
	commits := make([]parsedCommit, 0, len(subjects))
	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}

		parsed := parseCommit(subject)
		commits = append(commits, parsed)
	}
	return commits
}

func parseCommit(subject string) parsedCommit {
	matches := conventionalCommitPattern.FindStringSubmatch(subject)
	if len(matches) == 5 {
		commitType := strings.ToLower(strings.TrimSpace(matches[1]))
		scope := strings.ToLower(strings.TrimSpace(matches[2]))
		description := normalizeDescription(matches[4])
		return parsedCommit{
			Type:        commitType,
			Description: description,
			Breaking:    matches[3] == "!",
			Section:     classifySection(commitType, scope, description),
		}
	}

	description := normalizeDescription(subject)
	return parsedCommit{
		Type:        "other",
		Description: description,
		Section:     classifySection("other", "", description),
	}
}

func normalizeDescription(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	raw = strings.TrimSuffix(raw, ".")
	if raw == "" {
		return raw
	}

	runes := []rune(raw)
	if unicode.IsLetter(runes[0]) {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func classifySection(commitType, scope, description string) sectionKey {
	combined := strings.ToLower(strings.TrimSpace(scope + " " + description))
	scope = strings.ToLower(strings.TrimSpace(scope))
	description = strings.ToLower(strings.TrimSpace(description))

	if commitType == "docs" || containsAny(combined,
		"readme", "quickstart", "roadmap", "product", "strategy", "site", "documentation",
	) {
		return sectionDocs
	}

	if containsAny(scope,
		"sourcecontrol", "github", "gitlab", "bitbucket", "azuredevops", "svn", "scm",
	) || containsAny(description,
		"provider-first", "provider backed", "provider-backed", "remote audit", "remote-first",
	) {
		return sectionSCM
	}

	if containsAny(combined,
		"install", "package", "npm", "publish", "release flow", "release asset",
		"package manager", "distribution",
	) {
		return sectionDistribution
	}

	if containsAny(combined,
		"cli", "manifest", "inspect", "verify", "plan", "resolve", "doctor",
		"conflict", "ticket file", "ticket files", "release packet",
	) {
		return sectionCLI
	}

	switch commitType {
	case "feat", "fix", "refactor", "perf":
		return sectionCLI
	case "docs":
		return sectionDocs
	default:
		return sectionMaintenance
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func renderMarkdown(input Input, commits []parsedCommit) string {
	stats := buildStats(commits)
	sections := buildSections(commits)

	var b strings.Builder
	b.WriteString("## Summary\n\n")
	if stats.Total == 0 {
		b.WriteString(fmt.Sprintf("This `%s` release does not include any commit subjects in the selected range.\n", fallbackCurrentTag(input.CurrentTag)))
		return b.String()
	}

	b.WriteString(renderSummarySentence(input.CurrentTag, sections))
	b.WriteString("\n\n")

	summaryLines := []string{
		fmt.Sprintf("Changes captured: `%d`", stats.Total),
		fmt.Sprintf("Product updates: `%d`", stats.Product),
		fmt.Sprintf("Fixes: `%d`", stats.Fixes),
		fmt.Sprintf("Docs updates: `%d`", stats.Docs),
		fmt.Sprintf("Maintenance updates: `%d`", stats.Maintenance),
	}
	if stats.Breaking > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("Breaking changes: `%d`", stats.Breaking))
	}
	b.WriteString(renderList(summaryLines))
	b.WriteString("\n")

	if len(sections) > 0 {
		b.WriteString("## Highlights\n\n")
		for _, section := range orderedSections() {
			items := sections[section]
			if len(items) == 0 {
				continue
			}

			b.WriteString(fmt.Sprintf("### %s\n", sectionTitle(section)))
			b.WriteString(renderList(items))
			b.WriteString("\n")
		}
	}

	if upgradeNotes := buildUpgradeNotes(commits); len(upgradeNotes) > 0 {
		b.WriteString("## Upgrade Notes\n\n")
		b.WriteString(renderList(upgradeNotes))
		b.WriteString("\n")
	}

	if compareLine := renderCompareLine(input.PreviousTag, input.CurrentTag, input.CompareURL); compareLine != "" {
		b.WriteString("## Full Changelog\n\n")
		b.WriteString(compareLine)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func buildStats(commits []parsedCommit) summaryStats {
	stats := summaryStats{Total: len(commits)}
	for _, commit := range commits {
		switch commit.Type {
		case "feat", "refactor", "perf":
			stats.Product++
		case "fix":
			stats.Fixes++
		case "docs":
			stats.Docs++
		default:
			stats.Maintenance++
		}
		if commit.Breaking {
			stats.Breaking++
		}
	}
	return stats
}

func buildSections(commits []parsedCommit) map[sectionKey][]string {
	sections := map[sectionKey][]string{}
	seen := map[sectionKey]map[string]struct{}{}

	for _, commit := range commits {
		if commit.Description == "" {
			continue
		}
		if _, ok := seen[commit.Section]; !ok {
			seen[commit.Section] = map[string]struct{}{}
		}
		if _, ok := seen[commit.Section][commit.Description]; ok {
			continue
		}

		seen[commit.Section][commit.Description] = struct{}{}
		sections[commit.Section] = append(sections[commit.Section], commit.Description)
	}

	return sections
}

func orderedSections() []sectionKey {
	return []sectionKey{
		sectionCLI,
		sectionSCM,
		sectionDistribution,
		sectionDocs,
		sectionMaintenance,
	}
}

func sectionTitle(section sectionKey) string {
	switch section {
	case sectionCLI:
		return "CLI and release workflows"
	case sectionSCM:
		return "Source-control-native access"
	case sectionDistribution:
		return "Packaging and release automation"
	case sectionDocs:
		return "Docs and guidance"
	default:
		return "Engineering and maintenance"
	}
}

func renderSummarySentence(currentTag string, sections map[sectionKey][]string) string {
	focusAreas := make([]string, 0, 3)
	for _, section := range []sectionKey{sectionCLI, sectionSCM, sectionDistribution, sectionDocs, sectionMaintenance} {
		if len(sections[section]) == 0 {
			continue
		}
		focusAreas = append(focusAreas, summaryLabel(section))
		if len(focusAreas) == 3 {
			break
		}
	}

	if len(focusAreas) == 0 {
		return fmt.Sprintf("This `%s` release keeps `%s` moving forward with incremental maintenance work.", fallbackCurrentTag(currentTag), "gig")
	}

	return fmt.Sprintf("This `%s` release focuses on %s.", fallbackCurrentTag(currentTag), joinWithAnd(focusAreas))
}

func summaryLabel(section sectionKey) string {
	switch section {
	case sectionCLI:
		return "CLI and release workflows"
	case sectionSCM:
		return "source-control-native access"
	case sectionDistribution:
		return "packaging and release automation"
	case sectionDocs:
		return "docs and guidance"
	default:
		return "engineering and maintenance"
	}
}

func joinWithAnd(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	case 2:
		return values[0] + " and " + values[1]
	default:
		return strings.Join(values[:len(values)-1], ", ") + ", and " + values[len(values)-1]
	}
}

func renderList(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func buildUpgradeNotes(commits []parsedCommit) []string {
	notes := make([]string, 0)
	seen := map[string]struct{}{}
	for _, commit := range commits {
		if !commit.Breaking || commit.Description == "" {
			continue
		}
		if _, ok := seen[commit.Description]; ok {
			continue
		}
		seen[commit.Description] = struct{}{}
		notes = append(notes, commit.Description)
	}
	return notes
}

func renderCompareLine(previousTag, currentTag, compareURL string) string {
	if strings.TrimSpace(previousTag) == "" || strings.TrimSpace(currentTag) == "" {
		return ""
	}

	label := fmt.Sprintf("%s...%s", previousTag, currentTag)
	if strings.TrimSpace(compareURL) == "" {
		return fmt.Sprintf("- Range: `%s`", label)
	}
	return fmt.Sprintf("- Compare: [%s](%s)", label, compareURL)
}

func fallbackCurrentTag(currentTag string) string {
	currentTag = strings.TrimSpace(currentTag)
	if currentTag == "" {
		return "current release"
	}
	return currentTag
}

func renderInitialRelease(repositoryName string) string {
	return fmt.Sprintf(`## Summary

`+"`%s`"+` ships as a remote-first release audit CLI that helps teams answer one question before code moves forward:

**Did we miss any change for this ticket?**

## Why it matters

Release risk usually appears in follow-up fixes, multi-repository tickets, and last-minute manual steps that get reconstructed from memory. This first release packages the core read-only workflow into one CLI so teams can inspect, verify, plan, and document promotion scope before they touch production branches.

## What ships in this release

- Ticket-aware commit discovery across repositories
- Branch-to-branch verification with `+"`safe`"+`, `+"`warning`"+`, and `+"`blocked`"+` verdicts
- Promotion planning and Markdown or JSON release packets
- Direct remote inspection for supported source-control providers
- Optional AI briefings built from the same deterministic release evidence

## Notes

`+"`%s`"+` stays intentionally read-only for release actions. It improves visibility, release reasoning, and communication first so teams can automate with better evidence later.
`, repositoryName, repositoryName)
}
