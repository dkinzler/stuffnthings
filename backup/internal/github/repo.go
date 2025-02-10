package github

import (
	"backup/internal/exec"
	"backup/internal/fs"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type repo struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	CloneUrl string `json:"clone_url"`
	Private  bool   `json:"private"`
}

// implement the Item interface from the bubbles/list package
func (r repo) FilterValue() string {
	return r.Name
}

type loadReposResult struct {
	repos []repo
	err   error
}

func loadRepos(token string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: time.Second * 10}

		var repos []repo

		// we will get the complete url including query parameters for the next page from the last response header
		initialUrl, err := url.Parse("https://api.github.com/user/repos")
		if err != nil {
			log.Println("could not parse url:", err)
			return loadReposResult{err: err}
		}
		q := initialUrl.Query()
		// only get your own repos
		q.Add("affiliation", "owner")
		q.Add("per_page", "100")
		initialUrl.RawQuery = q.Encode()

		currentUrl := initialUrl.String()

		for currentUrl != "" {
			req, err := http.NewRequest("GET", currentUrl, nil)
			if err != nil {
				return loadReposResult{err: err}
			}
			req.Header.Add("Accept", "application/vnd.github+json")
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			// could be omitted to always use most recent version
			req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

			r, err, nextUrl := doRequest(client, req)
			if err != nil {
				return loadReposResult{err: err}
			}

			repos = append(repos, r...)
			currentUrl = nextUrl
		}

		return loadReposResult{repos: repos}
	}
}

func doRequest(client *http.Client, req *http.Request) ([]repo, error, string) {
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err, ""
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %v", resp.Status), ""
	}

	var repos []repo
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&repos)
	if err != nil {
		return nil, err, ""
	}

	var nextUrl string
	if link := resp.Header.Get("link"); link != "" {
		// extract url for next page from link field in response header
		// example: <https://api.github.com/user/repos?page=1>; rel="prev", <https://api.github.com/user/repos?page=1>; rel="last", <https://api.github.com/user/repos?page=1>; rel="first"
		// there is probably a more elegant way to do this, but it works for now
		i := strings.Index(link, "rel=\"next\"")
		if i != -1 {
			endIndex := -1
			startIndex := -1
			for j := i - 1; j >= 0; j-- {
				if link[j] == '>' {
					endIndex = j
				} else if link[j] == '<' {
					startIndex = j
					break
				}
			}
			if endIndex != -1 && startIndex != -1 && startIndex < endIndex {
				nextUrl = link[startIndex+1 : endIndex]
			}
		}
	}

	return repos, nil, nextUrl
}

type cloneRepoResult struct {
	id  int
	err error
}

func cloneRepo(repo repo, dir string, token string) tea.Cmd {
	// another approach would be to pass username/token in the url instead of via stdin e.g. https://username:token@example.com
	// these commands should not get logged in your shell history
	cmd := []string{"git", "clone", repo.CloneUrl, fs.JoinPath(dir, repo.Name)}
	opts := []exec.Option{
		exec.WithStdin(fmt.Sprintf("%s\n%s\n", repo.Owner.Login, token)),
	}

	return exec.Background(cmd, func(r exec.Result) tea.Msg {
		if r.ExitCode == 0 {
			return cloneRepoResult{id: repo.Id, err: nil}
		} else {
			if r.Err != nil {
				log.Println("error executing git clone:", r.Err)
				return cloneRepoResult{id: repo.Id, err: r.Err}
			} else {
				log.Printf("git clone failed, exit code: %v\nstdout: %s\nstderr: %s", r.ExitCode, r.Stdout, r.Stderr)
				return cloneRepoResult{id: repo.Id, err: fmt.Errorf("git clone failed with exit code %v", r.ExitCode)}
			}
		}
	}, opts...)
}
