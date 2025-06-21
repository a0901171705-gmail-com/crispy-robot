// TODO: rename this package to avoid clash with stdlib
package context

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/iostreams"
)

// Cap the number of git remotes to look up, since the user might have an
// unusually large number of git remotes.
const defaultRemotesForLookup = 5

func BaseRepo(apiClient *api.Client, remotes Remotes, io *iostreams.IOStreams) (ghrepo.Interface, error) {
	sort.Stable(remotes)

	if len(remotes) == 0 {
		return nil, errors.New("no git remotes")
	}

	// if any of the remotes already has a resolution, respect that
	for _, r := range remotes {
		if r.Resolved == "base" {
			return r, nil
		} else if r.Resolved != "" {
			repo, err := ghrepo.FromFullName(r.Resolved)
			if err != nil {
				return nil, err
			}
			return ghrepo.NewWithHost(repo.RepoOwner(), repo.RepoName(), r.RepoHost()), nil
		}
	}

	if !io.CanPrompt() {
		// we cannot prompt, so just resort to the 1st remote
		return remotes[0], nil
	}

	repos, err := NetworkRepos(apiClient, remotes, defaultRemotesForLookup)
	if err != nil {
		return nil, err
	}

	if len(repos) == 0 {
		return remotes[0], nil
	} else if len(repos) == 1 {
		return repos[0], nil
	}

	cs := io.ColorScheme()

	fmt.Fprintf(io.ErrOut,
		"%s No default remote repository has been set for this directory.\n",
		cs.FailureIcon())

	fmt.Fprintln(io.Out)

	return nil, errors.New(
		"please run `gh repo set-default` to select a default remote repository.")
}

// NetworkRepos fetches info about remotes for the network of repos.
// Pass a value of 0 to fetch info on all remotes.
func NetworkRepos(apiClient *api.Client, remotes Remotes, remotesForLookup int) ([]*api.Repository, error) {
	network, err := networkForRemotes(apiClient, remotes)
	if err != nil {
		return nil, err
	}

	var repos []*api.Repository
	repoMap := map[string]bool{}

	add := func(r *api.Repository) {
		fn := ghrepo.FullName(r)
		if _, ok := repoMap[fn]; !ok {
			repoMap[fn] = true
			repos = append(repos, r)
		}
	}

	for _, repo := range network.Repositories {
		if repo == nil {
			continue
		}
		if repo.Parent != nil {
			add(repo.Parent)
		}
		add(repo)
	}

	return repos, nil
}

func HeadRepos(apiClient *api.Client, remotes Remotes) ([]*api.Repository, error) {
	network, err := networkForRemotes(apiClient, remotes)
	if err != nil {
		return nil, err
	}

	var results []*api.Repository
	var ids []string // Check if repo duplicates
	for _, repo := range network.Repositories {
		if repo != nil && repo.ViewerCanPush() && !slices.Contains(ids, repo.ID) {
			results = append(results, repo)
			ids = append(ids, repo.ID)
		}
	}
	return results, nil
}

func networkForRemotes(apiClient *api.Client, remotes Remotes) (api.RepoNetworkResult, error) {
	sort.Stable(remotes)

	var reposToLookup []ghrepo.Interface
	for _, r := range remotes {
		reposToLookup = append(reposToLookup, r)
		if len(reposToLookup) == defaultRemotesForLookup {
			break
		}
	}

	return api.RepoNetwork(apiClient, reposToLookup)
}

// RemoteForRepo finds the git remote that points to a repository
func RemoteForRepo(repo ghrepo.Interface, remotes Remotes) (*Remote, error) {
	for _, remote := range remotes {
		if ghrepo.IsSame(remote, repo) {
			return remote, nil
		}
	}
	return nil, errors.New("not found")
}
