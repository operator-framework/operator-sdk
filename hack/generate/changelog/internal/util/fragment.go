package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type Fragment struct {
	Entries []FragmentEntry `yaml:"entries"`
}

func (f *Fragment) Validate() error {
	for i, e := range f.Entries {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("entry[%d] invalid: %v", i, err)
		}
	}
	return nil
}

type FragmentEntry struct {
	Description string          `json:"description"`
	Kind        EntryKind       `json:"kind"`
	Breaking    bool            `json:"breaking"`
	Migration   *EntryMigration `json:"migration,omitempty"`
	PullRequest *uint           `json:"pull_request_override,omitempty"`

	PullRequestLink string `json:"-"`
}

func (e *FragmentEntry) Validate() error {
	if err := e.Kind.Validate(); err != nil {
		return fmt.Errorf("invalid kind: %v", err)
	}

	if len(e.Description) == 0 {
		return errors.New("missing description")
	}

	if e.Breaking && e.Kind != Change && e.Kind != Removal {
		return fmt.Errorf("breaking changes can only be kind %q or %q, got %q", Change, Removal, e.Kind)
	}

	if e.Breaking && e.Migration == nil {
		return fmt.Errorf("breaking changes require migration sections")
	}

	if e.Migration != nil {
		if err := e.Migration.Validate(); err != nil {
			return fmt.Errorf("invalid migration: %v", err)
		}
	}
	return nil
}

func (e FragmentEntry) pullRequestLink(repo string) string {
	if e.PullRequest == nil {
		return ""
	}
	return fmt.Sprintf("[#%d](https://%s/pull/%d)", *e.PullRequest, repo, *e.PullRequest)
}

type EntryKind string

const (
	Addition    EntryKind = "addition"
	Change      EntryKind = "change"
	Removal     EntryKind = "removal"
	Deprecation EntryKind = "deprecation"
	Bugfix      EntryKind = "bugfix"
)

func (k EntryKind) Validate() error {
	for _, t := range []EntryKind{Addition, Change, Removal, Deprecation, Bugfix} {
		if k == t {
			return nil
		}
	}
	return fmt.Errorf("%q is not a supported kind", k)
}

type EntryMigration struct {
	Header string `yaml:"header"`
	Body   string `yaml:"body"`
}

func (m EntryMigration) Validate() error {
	if len(m.Header) == 0 {
		return errors.New("header not specified")
	}
	if len(m.Body) == 0 {
		return errors.New("body not specified")
	}
	return nil
}

func LoadEntries(fragmentsDir, repo string) ([]FragmentEntry, error) {
	files, err := ioutil.ReadDir(fragmentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read fragments directory: %w", err)
	}

	var entries []FragmentEntry
	for _, fragFile := range files {
		if fragFile.Name() == "00-template.yaml" {
			continue
		}
		if fragFile.IsDir() {
			log.Warnf("Skipping directory %q", fragFile.Name())
			continue
		}
		if filepath.Ext(fragFile.Name()) != ".yaml" && filepath.Ext(fragFile.Name()) != ".yml" {
			log.Warnf("Skipping non-YAML file %q", fragFile.Name())
			continue
		}
		path := filepath.Join(fragmentsDir, fragFile.Name())
		fragmentData, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read fragment file %q: %w", fragFile.Name(), err)
		}

		fragment := Fragment{}
		if err := yaml.Unmarshal(fragmentData, &fragment); err != nil {
			return nil, fmt.Errorf("failed to parse fragment file %q: %w", fragFile.Name(), err)
		}

		if err := fragment.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate fragment file %q: %w", fragFile.Name(), err)
		}

		prNum, err := prGetter.GetPullRequestNumberFor(path)
		if err != nil {
			log.Warn(err)
		}

		if prNum != 0 {
			for i, e := range fragment.Entries {
				if e.PullRequest == nil {
					fragment.Entries[i].PullRequest = &prNum
				}
			}
		}

		for i, e := range fragment.Entries {
			fragment.Entries[i].PullRequestLink = e.pullRequestLink(repo)
		}

		entries = append(entries, fragment.Entries...)
	}
	return entries, nil
}

var prGetter PullRequestNumberGetter = &gitPullRequestNumberGetter{}

type PullRequestNumberGetter interface {
	GetPullRequestNumberFor(file string) (uint, error)
}

type gitPullRequestNumberGetter struct{}

func (g *gitPullRequestNumberGetter) GetPullRequestNumberFor(filename string) (uint, error) {
	msg, err := g.getCommitMessage(filename)
	if err != nil {
		return 0, err
	}
	return g.parsePRNumber(msg)
}

func (g *gitPullRequestNumberGetter) getCommitMessage(filename string) (string, error) {
	args := fmt.Sprintf("log --follow --pretty=format:%%s --diff-filter=A --find-renames=90%% %s", filename)
	line, err := exec.Command("git", strings.Split(args, " ")...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to locate git commit for PR discovery: %v", err)
	}
	return string(line), nil
}

var numRegex = regexp.MustCompile(`\(#(\d+)\)$`)

func (g *gitPullRequestNumberGetter) parsePRNumber(msg string) (uint, error) {
	matches := numRegex.FindAllStringSubmatch(msg, 1)
	if len(matches) == 0 || len(matches[0]) < 2 {
		return 0, fmt.Errorf("could not find PR number in commit message")
	}
	u64, err := strconv.ParseUint(matches[0][1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PR number %q from commit message: %v", matches[0][1], err)
	}
	return uint(u64), nil
}
