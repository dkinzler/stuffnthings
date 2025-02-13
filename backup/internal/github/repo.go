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

type Repo struct {
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
func (r Repo) FilterValue() string {
	return r.Name
}

type loadReposResult struct {
	repos []Repo
	err   error
}

func loadReposCmd(token string) tea.Cmd {
	return func() tea.Msg {
		repos, err := LoadRepos(token)
		return loadReposResult{repos: repos, err: err}
	}
}

func LoadRepos(token string) ([]Repo, error) {
	var repos []Repo

	client := &http.Client{Timeout: time.Second * 10}

	// we will get the complete url including query parameters for the next page from the last response header
	initialUrl, err := url.Parse("https://api.github.com/user/repos")
	if err != nil {
		log.Println("could not parse url:", err)
		return repos, err
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
			return repos, err
		}
		req.Header.Add("Accept", "application/vnd.github+json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		// could be omitted to always use most recent version
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

		r, err, nextUrl := doRequest(client, req)
		if err != nil {
			return repos, err
		}

		repos = append(repos, r...)
		currentUrl = nextUrl
	}

	return repos, nil
}

func doRequest(client *http.Client, req *http.Request) ([]Repo, error, string) {
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err, ""
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %v", resp.Status), ""
	}

	var repos []Repo
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

func cloneRepoCmd(repo Repo, dir string, token string) tea.Cmd {
	return func() tea.Msg {
		r := CloneRepo(repo, dir, token)
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
	}
}

// Use GitHub CLI to clone repo.
// Alternatively, could use git clone directly but it would be less secure.
// We would have to pass the token via the clone url which will get stored in .git/config and
// would also be visible with "ps" while the clone operation is running.
// Afterwards we would need to update the origin url in .git/config to not include the token.
// Or we would need to setup a git credential helper to provide the token to git,
// but that would also get quite invovled.
func CloneRepo(repo Repo, dir string, token string) exec.Result {
	cmd := []string{"gh", "repo", "clone", repo.CloneUrl, fs.JoinPath(dir, repo.Name)}
	opts := []exec.Option{
		exec.WithTimeout(time.Second * 120),
		exec.WithEnv("GITHUB_TOKEN", token),
	}
	return exec.Background(cmd, opts...)
}

// func CloneRepo(repo Repo, dir string, token string) exec.Result {
// 	// these commands should not get logged in your shell history
// 	repoDir := fs.JoinPath(dir, repo.Name)
// 	gitCmd := fmt.Sprintf("git clone %s %s", buildCloneURL(repo, token), repoDir)
// 	cdCmd := fmt.Sprintf("cd %s", repoDir)
// 	setUrlCmd := fmt.Sprintf("git remote set-url origin %s", repo.CloneUrl)
// 	cmd := []string{"sh", "-c", fmt.Sprintf("%s && %s && %s", gitCmd, cdCmd, setUrlCmd)}
// 	opts := []exec.Option{
// 		exec.WithTimeout(time.Second * 120),
// 	}
// 	return exec.Background(cmd, opts...)
// }

// func buildCloneURL(repo Repo, token string) string {
// 	u, err := url.Parse(repo.CloneUrl)
// 	if err != nil {
// 		// TODO what to do here, return an error? can then include that in exec.Result?
// 		panic(err)
// 	}
// 	u.User = url.User(token)
// 	return u.String()
// }
