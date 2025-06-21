package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/cli/v2/internal/ghinstance"
)

const (
	defaultPerPage = 100
	orderKey       = "order"
	sortKey        = "sort"
)

var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
var pageRE = regexp.MustCompile(`(\?|&)page=(\d*)`)
var jsonTypeRE = regexp.MustCompile(`[/+]json($|;)`)

//go:generate moq -rm -out searcher_mock.go . Searcher
type Searcher interface {
	Code(Query) (CodeResult, error)
	Commits(Query) (CommitsResult, error)
	Repositories(Query) (RepositoriesResult, error)
	Issues(Query) (IssuesResult, error)
	URL(Query) string
}

type searcher struct {
	client *http.Client
	host   string
}

type httpError struct {
	Errors     []httpErrorItem
	Message    string
	RequestURL *url.URL
	StatusCode int
}

type httpErrorItem struct {
	Code     string
	Field    string
	Message  string
	Resource string
}

func NewSearcher(client *http.Client, host string) Searcher {
	return &searcher{
		client: client,
		host:   host,
	}
}

func (s searcher) Code(query Query) (CodeResult, error) {
	requestedLimit := query.Limit
	toRetrieve := requestedLimit

	var resp *http.Response
	var err error

	result := CodeResult{}

	for {
		// If we don't need any more results, we're done
		if toRetrieve == 0 {
			break
		}

		// If there are no further pages, we're out of results, so we're done
		pageNumber, hasNextPage := nextPage(resp)
		if !hasNextPage {
			break
		}
		query.Page = pageNumber

		// We will request either the limit if it's less than 1 page, or our default page
		// size. This means that for result sets larger than our default page size, we will
		// have a stable cursor offset.
		perPage := min(requestedLimit, defaultPerPage)
		query.Limit = perPage

		var page CodeResult
		resp, err = s.search(query, &page)
		if err != nil {
			// Return whatever results have been aggregated so far.
			// TODO: investigate whether anyone actually uses this, or if it
			// is just unidiomatic.
			return result, err
		}

		result.IncompleteResults = page.IncompleteResults
		result.Total = page.Total

		// If we're going to reach the requested limit, only add that many items,
		// otherwise add all the results.
		itemsToAdd := min(len(page.Items), toRetrieve)
		result.Items = append(result.Items, page.Items[:itemsToAdd]...)
		toRetrieve = toRetrieve - itemsToAdd
	}

	return result, nil
}

func (s searcher) Commits(query Query) (CommitsResult, error) {
	requestedLimit := query.Limit
	toRetrieve := requestedLimit

	var resp *http.Response
	var err error

	result := CommitsResult{}

	for {
		// If we don't need any more results, we're done
		if toRetrieve == 0 {
			break
		}

		// If there are no further pages, we're out of results, so we're done
		pageNumber, hasNextPage := nextPage(resp)
		if !hasNextPage {
			break
		}
		query.Page = pageNumber

		// We will request either the limit if it's less than 1 page, or our default page
		// size. This means that for result sets larger than our default page size, we will
		// have a stable cursor offset.
		perPage := min(requestedLimit, defaultPerPage)
		query.Limit = perPage

		var page CommitsResult
		resp, err = s.search(query, &page)
		if err != nil {
			// Return whatever results have been aggregated so far.
			// TODO: investigate whether anyone actually uses this, or if it
			// is just unidiomatic.
			return result, err
		}

		result.IncompleteResults = page.IncompleteResults
		result.Total = page.Total

		// If we're going to reach the requested limit, only add that many items,
		// otherwise add all the results.
		itemsToAdd := min(len(page.Items), toRetrieve)
		result.Items = append(result.Items, page.Items[:itemsToAdd]...)
		toRetrieve = toRetrieve - itemsToAdd
	}

	return result, nil
}

func (s searcher) Repositories(query Query) (RepositoriesResult, error) {
	requestedLimit := query.Limit
	toRetrieve := requestedLimit

	var resp *http.Response
	var err error

	result := RepositoriesResult{}

	for {
		// If we don't need any more results, we're done
		if toRetrieve == 0 {
			break
		}

		// If there are no further pages, we're out of results, so we're done
		pageNumber, hasNextPage := nextPage(resp)
		if !hasNextPage {
			break
		}
		query.Page = pageNumber

		// We will request either the limit if it's less than 1 page, or our default page
		// size. This means that for result sets larger than our default page size, we will
		// have a stable cursor offset.
		perPage := min(requestedLimit, defaultPerPage)
		query.Limit = perPage

		var page RepositoriesResult
		resp, err = s.search(query, &page)
		if err != nil {
			// Return whatever results have been aggregated so far.
			// TODO: investigate whether anyone actually uses this, or if it
			// is just unidiomatic.
			return result, err
		}

		result.IncompleteResults = page.IncompleteResults
		result.Total = page.Total

		// If we're going to reach the requested limit, only add that many items,
		// otherwise add all the results.
		itemsToAdd := min(len(page.Items), toRetrieve)
		result.Items = append(result.Items, page.Items[:itemsToAdd]...)
		toRetrieve = toRetrieve - itemsToAdd
	}

	return result, nil
}

func (s searcher) Issues(query Query) (IssuesResult, error) {
	requestedLimit := query.Limit
	toRetrieve := requestedLimit

	var resp *http.Response
	var err error

	result := IssuesResult{}

	for {
		// If we don't need any more results, we're done
		if toRetrieve == 0 {
			break
		}

		// If there are no further pages, we're out of results, so we're done
		pageNumber, hasNextPage := nextPage(resp)
		if !hasNextPage {
			break
		}
		query.Page = pageNumber

		// We will request either the limit if it's less than 1 page, or our default page
		// size. This means that for result sets larger than our default page size, we will
		// have a stable cursor offset.
		perPage := min(requestedLimit, defaultPerPage)
		query.Limit = perPage

		var page IssuesResult
		resp, err = s.search(query, &page)
		if err != nil {
			// Return whatever results have been aggregated so far.
			// TODO: investigate whether anyone actually uses this, or if it
			// is just unidiomatic.
			return result, err
		}

		result.IncompleteResults = page.IncompleteResults
		result.Total = page.Total

		// If we're going to reach the requested limit, only add that many items,
		// otherwise add all the results.
		itemsToAdd := min(len(page.Items), toRetrieve)
		result.Items = append(result.Items, page.Items[:itemsToAdd]...)
		toRetrieve = toRetrieve - itemsToAdd
	}

	return result, nil
}

func (s searcher) search(query Query, result interface{}) (*http.Response, error) {
	path := fmt.Sprintf("%ssearch/%s", ghinstance.RESTPrefix(s.host), query.Kind)
	qs := url.Values{}
	qs.Set("page", strconv.Itoa(query.Page))
	qs.Set("per_page", strconv.Itoa(query.Limit))
	qs.Set("q", query.String())
	if query.Order != "" {
		qs.Set(orderKey, query.Order)
	}
	if query.Sort != "" {
		qs.Set(sortKey, query.Sort)
	}
	url := fmt.Sprintf("%s?%s", path, qs.Encode())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if query.Kind == KindCode {
		req.Header.Set("Accept", "application/vnd.github.text-match+json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return resp, handleHTTPError(resp)
	}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(result)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (s searcher) URL(query Query) string {
	path := fmt.Sprintf("https://%s/search", s.host)
	qs := url.Values{}
	qs.Set("type", query.Kind)
	qs.Set("q", query.String())
	if query.Order != "" {
		qs.Set(orderKey, query.Order)
	}
	if query.Sort != "" {
		qs.Set(sortKey, query.Sort)
	}
	url := fmt.Sprintf("%s?%s", path, qs.Encode())
	return url
}

func (err httpError) Error() string {
	if err.StatusCode != 422 || len(err.Errors) == 0 {
		return fmt.Sprintf("HTTP %d: %s (%s)", err.StatusCode, err.Message, err.RequestURL)
	}
	query := strings.TrimSpace(err.RequestURL.Query().Get("q"))
	return fmt.Sprintf("Invalid search query %q.\n%s", query, err.Errors[0].Message)
}

func handleHTTPError(resp *http.Response) error {
	httpError := httpError{
		RequestURL: resp.Request.URL,
		StatusCode: resp.StatusCode,
	}
	if !jsonTypeRE.MatchString(resp.Header.Get("Content-Type")) {
		httpError.Message = resp.Status
		return httpError
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &httpError); err != nil {
		return err
	}
	return httpError
}

func nextPage(resp *http.Response) (int, bool) {
	if resp == nil {
		return 1, true
	}
	for _, m := range linkRE.FindAllStringSubmatch(resp.Header.Get("Link"), -1) {
		if !(len(m) > 2 && m[2] == "next") {
			continue
		}
		p := pageRE.FindStringSubmatch(m[1])
		if len(p) == 3 {
			i, err := strconv.Atoi(p[2])
			if err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
