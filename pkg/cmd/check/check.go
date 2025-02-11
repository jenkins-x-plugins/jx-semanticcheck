package check

import (
	"fmt"

	"github.com/jenkins-x-plugins/jx-semanticcheck/pkg/helpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"strings"

	"github.com/spf13/cobra"
)

// Options contains the command line flags
type Options struct {
	options.BaseOptions

	GitClient     gitclient.Interface
	CommandRunner cmdrunner.CommandRunner

	Dir string
}

var (
	cmdLong = templates.LongDesc(`
		Checks whether the commit messages in a pull request follow the Conventional Commits specification
`)

	cmdExample = templates.Examples(`
		jx-semanticcheck check 
`)

	ConventionalCommitTypes = []string{"feat", "fix", "perf", "refactor", "docs", "test", "revert", "style", "chore", "build"}
)

// NewCmdCheckSemantics creates a command object for the command
func NewCmdCheckSemantics() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "check",
		Short:   "Checks for whether the commits in a PR are Conventional Commits",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "Dir", "", "", "the directory of the repository")

	return cmd, o
}

func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate: %w", err)
	}

	commits, err := helpers.GetNewCommits(o.GitClient, o.Dir)
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	var failedCommits int
	for _, commit := range commits {
		var terminalMessage string
		indicator := "✓"

		if !isCommitConventional(commit.Message) {
			indicator = "x"
			terminalMessage = commit.Message
			failedCommits++
		}

		log.Logger().Infof("---  %s | %s --- %s\n"+
			"%s",
			commit.SHA, commit.Date, indicator, terminalMessage)
	}

	if failedCommits > 0 {
		return fmt.Errorf("%d commit(s) did not follow https://conventionalcommits.org/", failedCommits)
	}

	log.Logger().Infof("\nAll commits follow https://conventionalcommits.org/")
	return nil
}

// Validate checks that all the variables required to run are present
func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate base options: %w", err)
	}

	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return nil
}

// isCommitConventional checks whether a commit message follows the conventions by comparing its prefix
// to those in ConventionalCommitTypes
func isCommitConventional(commitMessage string) bool {
	commitMessage = strings.TrimSpace(strings.ToLower(commitMessage))

	// Ignore revert or merge commits
	if strings.Contains(commitMessage, "revert") || strings.Contains(commitMessage, "merge") {
		return true
	}

	idx := strings.Index(commitMessage, ":")
	if idx > 0 {
		commitType := commitMessage[0:idx]
		for _, conventionalType := range ConventionalCommitTypes {
			if strings.HasPrefix(commitType, conventionalType) {
				return true
			}
		}
	}
	return false
}
