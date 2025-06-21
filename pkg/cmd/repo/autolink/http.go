package autolink

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghinstance"
	"github.com/cli/cli/v2/internal/ghrepo"
)

type AutolinkGetter struct {
	HttpClient *http.Client
}

func NewAutolinkGetter(httpClient *http.Client) *AutolinkGetter {
	return &AutolinkGetter{
		HttpClient: httpClient,
	}
}

func (a *AutolinkGetter) Get(repo ghrepo.Interface) ([]autolink, error) {
	path := fmt.Sprintf("repos/%s/%s/autolinks", repo.RepoOwner(), repo.RepoName())
	url := ghinstance.RESTPrefix(repo.RepoHost()) + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("error getting autolinks: HTTP 404: Must have admin rights to Repository. (https://api.github.com/%s)", path)
	} else if resp.StatusCode > 299 {
		return nil, api.HandleHTTPError(resp)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var autolinks []autolink
	err = json.Unmarshal(b, &autolinks)
	if err != nil {
		return nil, err
	}

	return autolinks, nil
}
