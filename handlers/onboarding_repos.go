package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"git.commsnet.org/commstech/repository-detective/gitea"
)

func (h *OnboardingHandler) listOnboardRepositories(ctx context.Context, client *gitea.Client, reqOrgs []string) ([]gitea.RepositorySummary, error) {
	byKey := make(map[string]gitea.RepositorySummary)

	userRepos, err := client.ListAllUserRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user repositories: %w", err)
	}
	for _, repo := range userRepos {
		key := repoSummaryKey(repo)
		if key == "" {
			continue
		}
		byKey[key] = repo
	}

	for _, org := range mergeOnboardOrgList(reqOrgs, h.giteaScanOrgs) {
		orgRepos, err := client.ListAllOrgRepositories(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("list org %s repositories: %w", org, err)
		}
		for _, repo := range orgRepos {
			key := repoSummaryKey(repo)
			if key == "" {
				continue
			}
			byKey[key] = repo
		}
	}

	out := make([]gitea.RepositorySummary, 0, len(byKey))
	for _, repo := range byKey {
		out = append(out, repo)
	}
	sortRepositoriesForOnboard(out)
	return out, nil
}

func mergeOnboardOrgList(reqOrgs, configOrgs []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, org := range append(append([]string{}, reqOrgs...), configOrgs...) {
		org = strings.TrimSpace(org)
		if org == "" {
			continue
		}
		key := strings.ToLower(org)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, org)
	}
	return out
}

func repoSummaryKey(repo gitea.RepositorySummary) string {
	fullName := strings.TrimSpace(repo.FullName)
	if fullName == "" && strings.TrimSpace(repo.Name) != "" {
		owner := strings.TrimSpace(repo.Owner.Login)
		if owner != "" {
			fullName = owner + "/" + strings.TrimSpace(repo.Name)
		}
	}
	if fullName == "" {
		return ""
	}
	return strings.ToLower(fullName)
}

func sortRepositoriesForOnboard(repos []gitea.RepositorySummary) {
	sort.SliceStable(repos, func(i, j int) bool {
		di := isDogfoodRepository(repos[i])
		dj := isDogfoodRepository(repos[j])
		if di != dj {
			return di
		}
		return strings.ToLower(displayRepoFullName(repos[i])) < strings.ToLower(displayRepoFullName(repos[j]))
	})
}

func displayRepoFullName(repo gitea.RepositorySummary) string {
	if name := strings.TrimSpace(repo.FullName); name != "" {
		return name
	}
	owner := strings.TrimSpace(repo.Owner.Login)
	name := strings.TrimSpace(repo.Name)
	if owner != "" && name != "" {
		return owner + "/" + name
	}
	return name
}

func isDogfoodRepository(repo gitea.RepositorySummary) bool {
	_, repoName, ok := strings.Cut(strings.ToLower(displayRepoFullName(repo)), "/")
	if !ok {
		repoName = strings.ToLower(displayRepoFullName(repo))
	}
	switch repoName {
	case "repository-detective", "bugbot", "repository_detective":
		return true
	default:
		return strings.Contains(repoName, "repository-detective") || strings.Contains(repoName, "repository_detective")
	}
}
