package helpers

import (
	"fmt"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"

	"strings"
)

type Commit struct {
	SHA     string
	Message string
	Date    string
}

// GetNewCommits returns a list of Commit that have yet to be applied upstream
// from the current branch
func GetNewCommits(gitter gitclient.Interface, dir string) ([]*Commit, error) {
	defaultBranch, err := getDefaultBranchName(gitter, dir)
	if err != nil {
		return nil, err
	}

	// Gets a list of the commits on the current branch and whether they are new
	out, err := gitter.Command(dir, "cherry", defaultBranch)
	if err != nil {
		return nil, fmt.Errorf("running git: %w", err)
	}
	split := strings.Split(out, "\n")

	var newCommits []string
	for _, hash := range split {
		// Filter newCommits for commits that are present upstream
		if strings.Contains(hash, "+") {
			hash = strings.ReplaceAll(hash, "+ ", "")
			newCommits = append(newCommits, hash)
		}
	}

	return GetCommits(gitter, dir, newCommits)
}

func GetCommits(gitter gitclient.Interface, dir string, shas []string) ([]*Commit, error) {
	var commits []*Commit
	for _, sha := range shas {
		commit, err := GetCommit(gitter, dir, sha)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

// GetCommit uses "git show" to get information about a specific commit
func GetCommit(gitter gitclient.Interface, dir, sha string) (*Commit, error) {
	out, err := gitter.Command(dir, "show", "--quiet", sha,
		"--format=%H%n%s%n%ai")
	if err != nil {
		return nil, fmt.Errorf("running git: %w", err)
	}
	split := strings.Split(out, "\n")
	return &Commit{
		SHA:     split[0],
		Message: split[1],
		Date:    split[2],
	}, nil
}

func getDefaultBranchName(gitter gitclient.Interface, dir string) (string, error) {
	out, err := gitter.Command(dir, "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	if err != nil {
		return "", fmt.Errorf("running git: %w", err)
	}
	return out, nil
}
