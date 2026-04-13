package sourcecontrol

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"gig/internal/scm"
)

func ParseRepositoryTargets(spec string) ([]scm.Repository, error) {
	parts := strings.Split(spec, ",")
	repositories := make([]scm.Repository, 0, len(parts))
	seen := map[string]struct{}{}

	for _, part := range parts {
		repository, err := ParseRepositoryTarget(part)
		if err != nil {
			return nil, err
		}
		if repository.Root == "" {
			continue
		}
		if _, ok := seen[repository.Root]; ok {
			continue
		}
		seen[repository.Root] = struct{}{}
		repositories = append(repositories, repository)
	}

	if len(repositories) == 0 {
		return nil, fmt.Errorf("at least one repository target is required")
	}

	return repositories, nil
}

func ParseRepositoryTarget(raw string) (scm.Repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return scm.Repository{}, nil
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "github:"):
		return parseGitHubRepository(strings.TrimSpace(raw[len("github:"):]))
	case strings.HasPrefix(lower, "gitlab:"):
		return parseGitLabRepository(strings.TrimSpace(raw[len("gitlab:"):]))
	case strings.HasPrefix(lower, "bitbucket:"):
		return parseBitbucketRepository(strings.TrimSpace(raw[len("bitbucket:"):]))
	case strings.HasPrefix(lower, "azuredevops:"):
		return parseAzureDevOpsRepository(strings.TrimSpace(raw[len("azuredevops:"):]))
	case strings.HasPrefix(lower, "ado:"):
		return parseAzureDevOpsRepository(strings.TrimSpace(raw[len("ado:"):]))
	case strings.HasPrefix(lower, "https://github.com/"), strings.HasPrefix(lower, "http://github.com/"):
		u, err := url.Parse(raw)
		if err != nil {
			return scm.Repository{}, fmt.Errorf("parse repository target %q: %w", raw, err)
		}
		return parseGitHubRepository(strings.TrimPrefix(path.Clean(u.Path), "/"))
	case strings.HasPrefix(lower, "https://gitlab.com/"), strings.HasPrefix(lower, "http://gitlab.com/"):
		u, err := url.Parse(raw)
		if err != nil {
			return scm.Repository{}, fmt.Errorf("parse repository target %q: %w", raw, err)
		}
		return parseGitLabRepository(strings.TrimPrefix(path.Clean(u.Path), "/"))
	case strings.HasPrefix(lower, "https://bitbucket.org/"), strings.HasPrefix(lower, "http://bitbucket.org/"):
		u, err := url.Parse(raw)
		if err != nil {
			return scm.Repository{}, fmt.Errorf("parse repository target %q: %w", raw, err)
		}
		return parseBitbucketRepository(strings.TrimPrefix(path.Clean(u.Path), "/"))
	case strings.HasPrefix(lower, "https://dev.azure.com/"), strings.HasPrefix(lower, "http://dev.azure.com/"), strings.Contains(lower, ".visualstudio.com/"):
		return parseAzureDevOpsRepositoryURL(raw)
	default:
		return scm.Repository{}, fmt.Errorf("unsupported repository target %q", raw)
	}
}

func FormatScopeLabel(repositories []scm.Repository, fallback string) string {
	if len(repositories) == 0 {
		return fallback
	}
	labels := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		labels = append(labels, repository.Root)
	}
	return strings.Join(labels, ", ")
}

func parseGitHubRepository(identifier string) (scm.Repository, error) {
	identifier = strings.TrimSpace(identifier)
	identifier = strings.TrimPrefix(identifier, "/")
	identifier = strings.TrimSuffix(identifier, ".git")
	identifier = path.Clean(identifier)
	if identifier == "." || identifier == "" {
		return scm.Repository{}, fmt.Errorf("github repository target must be in owner/name form")
	}

	parts := strings.Split(identifier, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return scm.Repository{}, fmt.Errorf("github repository target must be in owner/name form")
	}

	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	root := fmt.Sprintf("github:%s/%s", owner, name)

	return scm.Repository{
		Name: name,
		Root: root,
		Type: scm.TypeGitHub,
	}, nil
}

func parseGitLabRepository(identifier string) (scm.Repository, error) {
	identifier = normalizeRemoteIdentifier(identifier)
	parts := strings.Split(identifier, "/")
	if len(parts) < 2 {
		return scm.Repository{}, fmt.Errorf("gitlab repository target must be in group/project form")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return scm.Repository{}, fmt.Errorf("gitlab repository target must be in group/project form")
		}
	}

	name := strings.TrimSpace(parts[len(parts)-1])
	root := fmt.Sprintf("gitlab:%s", strings.Join(parts, "/"))

	return scm.Repository{
		Name: name,
		Root: root,
		Type: scm.TypeGitLab,
	}, nil
}

func parseBitbucketRepository(identifier string) (scm.Repository, error) {
	identifier = normalizeRemoteIdentifier(identifier)
	parts := strings.Split(identifier, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return scm.Repository{}, fmt.Errorf("bitbucket repository target must be in workspace/repo form")
	}

	workspace := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	root := fmt.Sprintf("bitbucket:%s/%s", workspace, name)

	return scm.Repository{
		Name: name,
		Root: root,
		Type: scm.TypeBitbucket,
	}, nil
}

func parseAzureDevOpsRepository(identifier string) (scm.Repository, error) {
	identifier = normalizeRemoteIdentifier(identifier)
	parts := strings.Split(identifier, "/")
	if len(parts) != 3 {
		return scm.Repository{}, fmt.Errorf("azure devops repository target must be in org/project/repo form")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return scm.Repository{}, fmt.Errorf("azure devops repository target must be in org/project/repo form")
		}
	}

	root := fmt.Sprintf("azure-devops:%s/%s/%s", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]))

	return scm.Repository{
		Name: strings.TrimSpace(parts[2]),
		Root: root,
		Type: scm.TypeAzureDevOps,
	}, nil
}

func parseAzureDevOpsRepositoryURL(raw string) (scm.Repository, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return scm.Repository{}, fmt.Errorf("parse repository target %q: %w", raw, err)
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	trimmedPath := strings.TrimPrefix(path.Clean(u.Path), "/")
	parts := strings.Split(trimmedPath, "/")

	switch {
	case host == "dev.azure.com":
		if len(parts) >= 4 && strings.EqualFold(parts[2], "_git") {
			return parseAzureDevOpsRepository(strings.Join([]string{parts[0], parts[1], parts[3]}, "/"))
		}
	case strings.HasSuffix(host, ".visualstudio.com"):
		org := strings.TrimSuffix(host, ".visualstudio.com")
		if len(parts) >= 3 && strings.EqualFold(parts[1], "_git") {
			return parseAzureDevOpsRepository(strings.Join([]string{org, parts[0], parts[2]}, "/"))
		}
	}

	return scm.Repository{}, fmt.Errorf("azure devops repository target must match dev.azure.com/<org>/<project>/_git/<repo>")
}

func normalizeRemoteIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	identifier = strings.TrimPrefix(identifier, "/")
	identifier = strings.TrimSuffix(identifier, ".git")
	identifier = path.Clean(identifier)
	if identifier == "." {
		return ""
	}
	return identifier
}
