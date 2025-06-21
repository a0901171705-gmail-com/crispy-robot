package shared_test

import (
	"errors"
	"testing"

	ghContext "github.com/cli/cli/v2/context"
	"github.com/cli/cli/v2/pkg/cmd/secret/shared"
	"github.com/stretchr/testify/require"

	"github.com/cli/cli/v2/git"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/prompter"
)

func TestRequireNoAmbiguityBaseRepoFunc(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when there is only one remote", func(t *testing.T) {
		t.Parallel()

		// Given there is only one remote
		baseRepoFn := shared.RequireNoAmbiguityBaseRepoFunc(baseRepoStubFn, oneRemoteStubFn)

		// When fetching the base repo
		baseRepo, err := baseRepoFn()

		// It succeeds and returns the inner base repo
		require.NoError(t, err)
		require.True(t, ghrepo.IsSame(ghrepo.New("owner", "repo"), baseRepo))
	})

	t.Run("returns specific error when there are multiple remotes", func(t *testing.T) {
		t.Parallel()

		// Given there are multiple remotes
		baseRepoFn := shared.RequireNoAmbiguityBaseRepoFunc(baseRepoStubFn, twoRemotesStubFn)

		// When fetching the base repo
		_, err := baseRepoFn()

		// It succeeds and returns the inner base repo
		var multipleRemotesError shared.MultipleRemotesError
		require.ErrorAs(t, err, &multipleRemotesError)
		require.Equal(t, ghContext.Remotes{
			{
				Remote: &git.Remote{
					Name: "origin",
				},
				Repo: ghrepo.New("owner", "fork"),
			},
			{
				Remote: &git.Remote{
					Name: "upstream",
				},
				Repo: ghrepo.New("owner", "repo"),
			},
		}, multipleRemotesError.Remotes)
	})

	t.Run("when the remote fetching function fails, it returns the error", func(t *testing.T) {
		t.Parallel()

		// Given the remote fetching function fails
		baseRepoFn := shared.RequireNoAmbiguityBaseRepoFunc(baseRepoStubFn, errRemoteStubFn)

		// When fetching the base repo
		_, err := baseRepoFn()

		// It returns the error
		require.Equal(t, errors.New("test remote error"), err)
	})

	t.Run("when the wrapped base repo function fails, it returns the error", func(t *testing.T) {
		t.Parallel()

		// Given the wrapped base repo function fails
		baseRepoFn := shared.RequireNoAmbiguityBaseRepoFunc(errBaseRepoStubFn, oneRemoteStubFn)

		// When fetching the base repo
		_, err := baseRepoFn()

		// It returns the error
		require.Equal(t, errors.New("test base repo error"), err)
	})
}

func TestPromptWhenMultipleRemotesBaseRepoFunc(t *testing.T) {
	t.Parallel()

	t.Run("when there is no error from wrapped base repo func, then it succeeds without prompting", func(t *testing.T) {
		t.Parallel()

		// Given the base repo function succeeds
		baseRepoFn := shared.PromptWhenMultipleRemotesBaseRepoFunc(baseRepoStubFn, nil)

		// When fetching the base repo
		baseRepo, err := baseRepoFn()

		// It succeeds and returns the inner base repo
		require.NoError(t, err)
		require.True(t, ghrepo.IsSame(ghrepo.New("owner", "repo"), baseRepo))
	})

	t.Run("when the wrapped base repo func returns a specific error, then the prompter is used for disambiguation, with the resolved remote as the default", func(t *testing.T) {
		t.Parallel()

		pm := prompter.NewMockPrompter(t)
		pm.RegisterSelect(
			"Select a base repo",
			[]string{"owner/fork", "owner/repo"},
			func(_, def string, opts []string) (int, error) {
				t.Helper()
				require.Equal(t, "owner/repo", def)
				return prompter.IndexFor(opts, "owner/repo")
			},
		)

		// Given the wrapped base repo func returns a specific error
		baseRepoFn := shared.PromptWhenMultipleRemotesBaseRepoFunc(errMultipleRemotesStubFn, pm)

		// When fetching the base repo
		baseRepo, err := baseRepoFn()

		// It uses the prompter for disambiguation
		require.NoError(t, err)
		require.True(t, ghrepo.IsSame(ghrepo.New("owner", "repo"), baseRepo))
	})

	t.Run("when the prompter returns an error, then it is returned", func(t *testing.T) {
		t.Parallel()

		// Given the prompter returns an error
		pm := prompter.NewMockPrompter(t)
		pm.RegisterSelect(
			"Select a base repo",
			[]string{"owner/fork", "owner/repo"},
			func(_, _ string, opts []string) (int, error) {
				return 0, errors.New("test prompt error")
			},
		)

		// Given the wrapped base repo func returns a specific error
		baseRepoFn := shared.PromptWhenMultipleRemotesBaseRepoFunc(errMultipleRemotesStubFn, pm)

		// When fetching the base repo
		_, err := baseRepoFn()

		// It returns the error
		require.Equal(t, errors.New("test prompt error"), err)
	})

	t.Run("when the wrapped base repo func returns a non-specific error, then it is returned", func(t *testing.T) {
		t.Parallel()

		// Given the wrapped base repo func returns a non-specific error
		baseRepoFn := shared.PromptWhenMultipleRemotesBaseRepoFunc(errBaseRepoStubFn, nil)

		// When fetching the base repo
		_, err := baseRepoFn()

		// It returns the error
		require.Equal(t, errors.New("test base repo error"), err)
	})
}

func TestMultipleRemotesErrorMessage(t *testing.T) {
	err := shared.MultipleRemotesError{}
	require.EqualError(t, err, "multiple remotes detected. please specify which repo to use by providing the -R or --repo argument")
}

func errMultipleRemotesStubFn() (ghrepo.Interface, error) {
	remote1 := &ghContext.Remote{
		Remote: &git.Remote{
			Name: "origin",
		},
		Repo: ghrepo.New("owner", "fork"),
	}

	remote2 := &ghContext.Remote{
		Remote: &git.Remote{
			Name:     "upstream",
			Resolved: "base",
		},
		Repo: ghrepo.New("owner", "repo"),
	}

	return nil, shared.MultipleRemotesError{
		Remotes: ghContext.Remotes{
			remote1,
			remote2,
		},
	}
}

func baseRepoStubFn() (ghrepo.Interface, error) {
	return ghrepo.New("owner", "repo"), nil
}

func oneRemoteStubFn() (ghContext.Remotes, error) {
	remote := &ghContext.Remote{
		Remote: &git.Remote{
			Name: "origin",
		},
		Repo: ghrepo.New("owner", "repo"),
	}

	return ghContext.Remotes{
		remote,
	}, nil
}

func twoRemotesStubFn() (ghContext.Remotes, error) {
	remote1 := &ghContext.Remote{
		Remote: &git.Remote{
			Name: "origin",
		},
		Repo: ghrepo.New("owner", "fork"),
	}

	remote2 := &ghContext.Remote{
		Remote: &git.Remote{
			Name: "upstream",
		},
		Repo: ghrepo.New("owner", "repo"),
	}
	return ghContext.Remotes{
		remote1,
		remote2,
	}, nil
}

func errRemoteStubFn() (ghContext.Remotes, error) {
	return nil, errors.New("test remote error")
}

func errBaseRepoStubFn() (ghrepo.Interface, error) {
	return nil, errors.New("test base repo error")
}
