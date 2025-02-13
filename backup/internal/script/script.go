package script

import (
	"backup/internal/config"
	"backup/internal/exec"
	"backup/internal/fs"
	"backup/internal/github"
	"backup/internal/zip"
	"fmt"
	"strings"
)

func Backup(configFile string) {
	out.Println("loading config")
	config, err := config.LoadConfig(configFile)
	if err != nil {
		out.Println("error:", err)
		return
	}

	backupDir, ok := validateBackupDir(config.BackupDir)
	if !ok {
		return
	}

	backupGithub(backupDir, config.Github)

	zipDir(backupDir, config.Zip)
}

func validateBackupDir(backupDir string) (string, bool) {
	out.Println("validating backup directory:", backupDir)
	absPath, err := fs.AbsPath(backupDir)
	if err != nil {
		out.Println("error:", err)
		return "", false
	}

	exists, err := fs.DirExists(absPath)
	if err != nil {
		out.Println("error:", err)
		return "", false
	}

	if exists {
		out.Println("warning: backup directory is not empty, files might get overwritten")
		if !confirmPrompt("continue?") {
			return "", false
		}
	}

	exists, err = fs.DirExists(fs.ParentPath(absPath))
	if err != nil {
		out.Println("error:", err)
		return "", false
	}

	if !exists {
		out.Println("warning: parent directory does not exist, might be a typo")
		if !confirmPrompt("continue?") {
			return "", false
		}
	}

	return absPath, true
}

func backupGithub(backupDir string, config github.Config) {
	out.Println()
	out.Println("backing up github repos")

	err := exec.CommandAvailable("gh")
	if err != nil {
		out.Println("error: no valid gh (github cli) executable found:", err)
		return
	}

	backupDir = fs.JoinPath(backupDir, "github")

	if config.Token == "" {
		out.Println("personal access token not provided, update your config and try again")
		return
	}

	out.Println("loading repos")
	var repos []github.Repo
	for {
		repos, err = github.LoadRepos(config.Token)
		if err != nil {
			out.Println("error:", err)
			if !confirmPrompt("try again?") {
				break
			}
		} else {
			break
		}
	}

	if len(repos) == 0 {
		out.Println("no repos to clone")
		return
	}

	out.Println("found", len(repos), "repos to clone")

	reposToClone := repos
	for {
		var failed []github.Repo
		for i, repo := range reposToClone {
			out.Printf("cloning repo %s (%v/%v)\n", repo.FullName, i+1, len(reposToClone))
			cloneDir := fs.JoinPath(backupDir, repo.Name)
			empty, err := fs.IsDirEmpty(cloneDir)
			if err != nil {
				failed = append(failed, repo)
				out.Println("error: could not check if clone directory exists:", err)
			}
			if !empty {
				out.Println("skipping, target directory", cloneDir, "is not empty")
				continue
			}

			result := github.CloneRepo(repo, backupDir, config.Token)
			if result.Err != nil {
				failed = append(failed, repo)
				out.Println("error:", result.Err)
			} else if result.ExitCode != 0 {
				failed = append(failed, repo)
				out.Println("error: gh exited with code", result.ExitCode)
				if len(result.Stdout) > 0 {
					out.Println("stdout:")
					out.Println(result.Stdout)
				}
				if len(result.Stderr) > 0 {
					out.Println("stderr:")
					out.Println(result.Stderr)
				}
			}
		}

		if len(failed) > 0 {
			if len(failed) == 1 {
				out.Println("1 repo failed to clone")
			} else {
				out.Println(len(failed), "repos failed to clone")
			}
			if confirmPrompt("try again?") {
				reposToClone = failed
			} else {
				break
			}
		} else {
			break
		}
	}
}

func confirmPrompt(text string) bool {
	for {
		out.Printf("%s (y/n): ", text)
		response, err := in.ReadLine()
		if err != nil {
			panic(err)
		}
		response = strings.TrimSpace(response)

		if response == "y" {
			return true
		} else if response == "n" {
			return false
		}

		out.Println("invalid response")
	}
}

func zipDir(dir string, config zip.Config) {
	out.Println()
	out.Println("zipping")
	err := exec.CommandAvailable("zip")
	if err != nil {
		out.Println("error: no valid zip executable found:", err)
		return
	}

	if config.File == "" {
		out.Println("skipping, no zip file specified")
		return
	}

	filePath, err := fs.AbsPath(config.File)
	if err != nil {
		out.Println("error: invalid zip file:", err)
	}

	// copied from package zip
	base := fs.BasePath(dir)
	parent := fs.ParentPath(dir)
	// why change directory? because otherwise the zip file will contain all the parent directories of files
	// e.g. when you unzip you will get home/username/backup/somefile instead of just backup/somefile
	// sh starts a new shell, so we do not have to worry about changing directory back
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && zip -r %s %s", parent, filePath, base)}
	// no timeout, zip might take a while
	result := exec.Background(cmd, exec.WithTimeout(0))

	if result.Err != nil {
		out.Println("error: zip failed:", err)
	} else if result.ExitCode != 0 {
		out.Println("error: zip failed with exit code", result.ExitCode)
		if len(result.Stdout) > 0 {
			out.Println("stdout:")
			out.Println(result.Stdout)
		}
		if len(result.Stderr) > 0 {
			out.Println("stderr:")
			out.Println(result.Stderr)
		}
	}

	size, err := fs.FileSize(filePath)
	if err == nil {
		out.Println("created", filePath, fileSizeString(size))
	}
}

func fileSizeString(size int64) string {
	if size >= 1024*1024 {
		return fmt.Sprintf("%vM", size/(1024*1024))
	} else if size >= 1024 {
		return fmt.Sprintf("%vK", size/1024)
	} else {
		return fmt.Sprintf("%v", size)
	}
}
