package check_test

import (
	"fmt"
	"github.com/jenkins-x-plugins/jx-semanticcheck/pkg/cmd/check"
	"github.com/jenkins-x-plugins/jx-semanticcheck/pkg/helpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/stretchr/testify/assert"
	"testing"
)

type commitWrapper struct {
	helpers.Commit
	IsAdded bool
}

func TestCheckRun(t *testing.T) {
	testCases := []struct {
		testCase        string
		commitsToReturn []commitWrapper
		mainBranchName  string
		shouldError     bool
	}{
		{
			testCase: "semantic with fix and feat tags pass",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "fix: added missing index to database",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
				{
					Commit: helpers.Commit{
						SHA:     "a80b106e8e67583007de35c6211800f693abc364",
						Message: "feat: Added new table products",
						Date:    "2022-07-27 12:34:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "main",
			shouldError:    false,
		},
		{
			testCase: "semantic with chore tag pass on different main branch",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "chore: update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "main2",
			shouldError:    false,
		},
		{
			testCase: "non semantic fails",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "main",
			shouldError:    true,
		},
		{
			testCase: "non semantic sandwiched by semantics fails",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "chore: update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
				{
					Commit: helpers.Commit{
						SHA:     "b70b106e8e67583007de35c6211800f693abc364",
						Message: "feat: update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
				{
					Commit: helpers.Commit{
						SHA:     "c70b106e8e67583007de35c6211800f693abc364",
						Message: "update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
				{
					Commit: helpers.Commit{
						SHA:     "d70b106e8e67583007de35c6211800f693abc364",
						Message: "fix: update jx-helpers to v0.2.0",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "main",
			shouldError:    true,
		},
		{
			testCase: "revert",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "revert: revert something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "chore",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "chore: chore something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "feat",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "feat: feat something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "feat!",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "feat!: feat something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "refactor",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "refactor: refactor something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "docs",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "docs: docs something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "test",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "test: test something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "style",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "style: style something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "without colon",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "style something",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    true,
		},
		{
			testCase: "with scope",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "docs(README): improve readability",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "merge",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "Merge pull request #287",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "revert",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "Revert 'feat: added date offset so that we can",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
		{
			testCase: "build",
			commitsToReturn: []commitWrapper{
				{
					Commit: helpers.Commit{
						SHA:     "a70b106e8e67583007de35c6211800f693abc364",
						Message: "build: the service binaries",
						Date:    "2022-07-27 12:24:18 +0100",
					},
					IsAdded: true,
				},
			},
			mainBranchName: "master",
			shouldError:    false,
		},
	}

	mockGitClient := mockGitClient{}

	o := check.Options{
		BaseOptions:   options.BaseOptions{},
		GitClient:     &mockGitClient,
		CommandRunner: nil,
		Dir:           "some/test/Dir",
	}

	for _, tt := range testCases {
		// Setup the mock git client
		mockGitClient.reset()
		mockGitClient.addMockedCommand(o.Dir, fmt.Sprintf("origin/%s", tt.mainBranchName), nil, "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
		allCommits := buildCommitsToReturnList(tt.commitsToReturn)
		mockGitClient.addMockedCommand(o.Dir, allCommits, nil, "cherry", fmt.Sprintf("origin/%s", tt.mainBranchName))
		for _, commit := range tt.commitsToReturn {
			mockGitClient.addMockedCommand(o.Dir, commit.SHA+"\n"+commit.Message+"\n"+commit.Date, nil, "show", "--quiet", commit.SHA, "--format=%H%n%s%n%ai")
		}

		t.Run(tt.testCase, func(t *testing.T) {
			err := o.Run()
			if tt.shouldError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func buildCommitsToReturnList(commits []commitWrapper) string {
	var allCommits string

	for _, commit := range commits {
		plusOrMinus := ""
		if commit.IsAdded {
			plusOrMinus = "+"
		} else {
			plusOrMinus = "-"
		}

		allCommits = allCommits + plusOrMinus + " " + commit.SHA + "\n"
	}

	return allCommits
}
