package gits

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
)

type CommitInfo struct {
	Kind    string
	Feature string
	Message string
	group   *CommitGroup
}

type CommitGroup struct {
	Title string
	Order int
}

var (
	groupCounter = 0

	// ConventionalCommitTitles textual descriptions for
	// Conventional Commit types: https://conventionalcommits.org/
	ConventionalCommitTitles = map[string]*CommitGroup{
		"feat":     createCommitGroup("New Features"),
		"fix":      createCommitGroup("Bug Fixes"),
		"perf":     createCommitGroup("Performance Improvements"),
		"refactor": createCommitGroup("Code Refactoring"),
		"docs":     createCommitGroup("Documentation"),
		"test":     createCommitGroup("Tests"),
		"revert":   createCommitGroup("Reverts"),
		"style":    createCommitGroup("Styles"),
		"chore":    createCommitGroup("Chores"),
		"":         createCommitGroup(""),
	}

	unknownKindOrder = len(ConventionalCommitTitles) + 1
)

func createCommitGroup(title string) *CommitGroup {
	groupCounter++
	return &CommitGroup{
		Title: title,
		Order: groupCounter,
	}
}

// ConventionalCommitTypeToTitle returns the title of the conventional commit type
// see: https://conventionalcommits.org/

// ParseCommit parses a conventional commit
// see: https://conventionalcommits.org/
func ParseCommit(message string) *CommitInfo {
	answer := &CommitInfo{
		Message: message,
	}

	idx := strings.Index(message, ":")
	if idx > 0 {
		kind := message[0:idx]
		if strings.HasSuffix(kind, ")") {
			ix := strings.Index(kind, "(")
			if ix > 0 {
				answer.Feature = strings.TrimSpace(kind[ix+1 : len(kind)-1])
				kind = strings.TrimSpace(kind[0:ix])
			}
		}
		answer.Kind = kind
		rest := strings.TrimSpace(message[idx+1:])

		answer.Message = rest
	}
	return answer
}

func (c *CommitInfo) Group() *CommitGroup {
	if c.group == nil {
		c.group = ConventionalCommitTitles[strings.ToLower(c.Kind)]
	}
	return c.group
}

func (c *CommitInfo) Title() string {
	return c.Group().Title
}

func (c *CommitInfo) Order() int {
	return c.Group().Order
}

type GroupAndCommitInfos struct {
	group   *CommitGroup
	commits []string
}

// GenerateMarkdown generates the markdown document for the commits
func GenerateMarkdown(releaseSpec *v1.ReleaseSpec, gitInfo *giturl.GitRepository) (string, error) {
	var commitInfos []*CommitInfo

	groupAndCommits := map[int]*GroupAndCommitInfos{}

	issues := releaseSpec.Issues
	issueMap := map[string]*v1.IssueSummary{}
	for k := range issues {
		cp := issues[k]
		issueMap[cp.ID] = &cp
	}

	for _, cs := range releaseSpec.Commits {
		commits := cs
		message := commits.Message
		if message != "" {
			ci := ParseCommit(message)

			description := "* " + describeCommit(gitInfo, &commits, ci, issueMap) + "\n"
			group := ci.Group()
			if group != nil {
				gac := groupAndCommits[group.Order]
				if gac == nil {
					gac = &GroupAndCommitInfos{
						group:   group,
						commits: []string{},
					}
					groupAndCommits[group.Order] = gac
				}
				gac.commits = append(gac.commits, description)
			}
			commitInfos = append(commitInfos, ci)
		}
	}

	prs := releaseSpec.PullRequests

	var buffer bytes.Buffer
	if len(commitInfos) == 0 && len(issues) == 0 && len(prs) == 0 {
		return "", nil
	}

	buffer.WriteString("## Changes\n")

	hasTitle := false
	for i := 0; i <= unknownKindOrder; i++ {
		gac := groupAndCommits[i]
		if gac != nil && len(gac.commits) > 0 {
			group := gac.group
			if group != nil {
				legend := ""
				buffer.WriteString("\n")
				if group.Title == "" && hasTitle {
					group.Title = "Other Changes"
					legend = "These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:\n\n"
				}
				if group.Title != "" {
					hasTitle = true
					buffer.WriteString("### " + group.Title + "\n\n" + legend)
				}
			}
			previous := ""
			for _, msg := range gac.commits {
				if msg != previous {
					buffer.WriteString(msg)
					previous = msg
				}
			}
		}
	}

	if len(issues) > 0 {
		buffer.WriteString("\n### Issues\n\n")

		previous := ""
		for k := range issues {
			i := issues[k]
			msg := describeIssue(gitInfo, &i)
			if msg != previous {
				buffer.WriteString("* " + msg + "\n")
				previous = msg
			}
		}
	}
	if len(prs) > 0 {
		buffer.WriteString("\n### Pull Requests\n\n")

		previous := ""
		for k := range prs {
			pullRequest := prs[k]
			msg := describeIssue(gitInfo, &pullRequest)
			if msg != previous {
				buffer.WriteString("* " + msg + "\n")
				previous = msg
			}
		}
	}

	if len(releaseSpec.DependencyUpdates) > 0 {
		buffer.WriteString("\n### Dependency Updates\n\n")
		var previous v1.DependencyUpdate
		sequence := make([]v1.DependencyUpdate, 0)
		buffer.WriteString("| Dependency | Component | New Version | Old Version |\n")
		buffer.WriteString("| ---------- | --------- | ----------- | ----------- |\n")
		for i := range releaseSpec.DependencyUpdates {
			du := releaseSpec.DependencyUpdates[i]
			sequence = append(sequence, du)
			// If it's the last element, or if the owner/repo:component changes, then print - this logic relies of the sort
			// being owner, repo, component, fromVersion, ToVersion, which is done above
			if i == len(releaseSpec.DependencyUpdates)-1 || du.Owner != previous.Owner || du.Repo != previous.Repo || du.Component != previous.Component {
				// find the earliest from version
				fromDu := sequence[0]
				toDu := sequence[len(sequence)-1]
				msg := fmt.Sprintf("| [%s/%s](%s) | %s | [%s](%s) | [%s](%s)|\n", toDu.Owner, toDu.Repo, toDu.URL, toDu.Component, toDu.ToVersion, toDu.ToReleaseHTMLURL, fromDu.FromVersion, fromDu.FromReleaseHTMLURL)
				buffer.WriteString(msg)
				sequence = make([]v1.DependencyUpdate, 0)
			}
			previous = du
		}
	}
	return buffer.String(), nil
}

func describeIssue(info *giturl.GitRepository, issue *v1.IssueSummary) string {
	return describeIssueShort(issue) + issue.Title + describeUser(info, issue.User)
}

func describeIssueShort(issue *v1.IssueSummary) string {
	prefix := ""
	id := issue.ID
	if len(id) > 0 {
		// lets only add the hash prefix for numeric ids
		_, err := strconv.Atoi(id)
		if err == nil {
			prefix = "#"
		}
	}
	return "[" + prefix + issue.ID + "](" + issue.URL + ") "
}

func describeUser(info *giturl.GitRepository, user *v1.UserDetails) string {
	answer := ""
	if user != nil {
		userText := ""
		login := user.Login
		url := user.URL
		label := login
		if label == "" {
			label = user.Name
		}
		if url == "" && login != "" {
			url = stringhelpers.UrlJoin(info.HostURL(), login)
		}
		if url == "" {
			userText = label
		} else if label != "" {
			userText = "[" + label + "](" + url + ")"
		}
		if userText != "" {
			answer = " (" + userText + ")"
		}
	}
	return answer
}

func describeCommit(info *giturl.GitRepository, cs *v1.CommitSummary, ci *CommitInfo, issueMap map[string]*v1.IssueSummary) string {
	prefix := ""
	if ci.Feature != "" {
		prefix = ci.Feature + ": "
	}
	message := strings.TrimSpace(ci.Message)
	lines := strings.Split(message, "\n")

	// TODO add link to issue etc...
	user := cs.Author
	if user == nil {
		user = cs.Committer
	}
	issueText := ""
	for k := range cs.IssueIDs {
		issue := issueMap[cs.IssueIDs[k]]
		if issue != nil {
			issueText += " " + describeIssueShort(issue)
		}
	}
	return prefix + lines[0] + describeUser(info, user) + issueText
}
