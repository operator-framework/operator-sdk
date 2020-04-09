package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

var numRegex = regexp.MustCompile(`\(#(\d+)\)$`)

const repo = "github.com/operator-framework/operator-sdk"

func main() {
	var (
		tag           string
		fragmentsDir  string
		changelogFile string
		migrationDir  string
		validateOnly  bool
	)

	flag.StringVar(&tag, "tag", "",
		"Title for generated CHANGELOG and migration guide sections")
	flag.StringVar(&fragmentsDir, "fragments-dir", filepath.Join("changelog", "fragments"),
		"Path to changelog fragments directory")
	flag.StringVar(&changelogFile, "changelog", "CHANGELOG.md",
		"Path to CHANGELOG")
	flag.StringVar(&migrationDir, "migration-guide-dir",
		filepath.Join("website", "content", "en", "docs", "migration"),
		"Path to migration guide directory")
	flag.BoolVar(&validateOnly, "validate-only", false,
		"Only validate fragments")
	flag.Parse()

	if tag == "" && !validateOnly {
		log.Fatalf("flag '-tag' is required without '-validate-only'")
	}
	version, err := semver.Parse(strings.TrimPrefix(tag, "v"))
	if err != nil {
		log.Fatalf("flag '-tag' is not a valid semantic version: %v", err)
	}
	if len(version.Pre) > 0 || len(version.Build) > 0 {
		log.Fatalf("flag '-tag' must not include a build number or pre-release identifiers")
	}

	entries, err := loadEntries(fragmentsDir)
	if err != nil {
		log.Fatalf("failed to load fragments: %v", err)
	}
	if len(entries) == 0 {
		log.Fatalf("no Entries found")
	}

	if validateOnly {
		return
	}

	if err := updateChangelog(config{
		File:    changelogFile,
		Version: version,
		Entries: entries,
	}); err != nil {
		log.Fatalf("failed to update CHANGELOG: %v", err)
	}

	if err := createMigrationGuide(config{
		File:    filepath.Join(migrationDir, fmt.Sprintf("v%s.md", version)),
		Version: version,
		Entries: entries,
	}); err != nil {
		log.Fatalf("failed to create migration guide: %v", err)
	}
}

func loadEntries(fragmentsDir string) ([]entry, error) {
	files, err := ioutil.ReadDir(fragmentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read fragments directory: %w", err)
	}

	var entries []entry
	for _, fragFile := range files {
		if fragFile.Name() == "00-template.yaml" {
			continue
		}
		if fragFile.IsDir() {
			log.Warnf("Skipping directory %q", fragFile.Name())
			continue
		}
		if filepath.Ext(fragFile.Name()) != ".yaml" {
			log.Warnf("Skipping non-YAML file %q", fragFile.Name())
			continue
		}
		path := filepath.Join(fragmentsDir, fragFile.Name())
		fragmentData, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read fragment file %q: %w", fragFile.Name(), err)
		}

		fragment := fragment{}
		if err := yaml.Unmarshal(fragmentData, &fragment); err != nil {
			return nil, fmt.Errorf("failed to parse fragment file %q: %w", fragFile.Name(), err)
		}

		if err := fragment.validate(); err != nil {
			return nil, fmt.Errorf("failed to validate fragment file %q: %w", fragFile.Name(), err)
		}

		commitMsg, err := getCommitMessage(path)
		if err != nil {
			log.Warnf("failed to get commit message for fragment file %q: %v", fragFile.Name(), err)
		}
		prNum, err := parsePRNumber(commitMsg)
		if err != nil {
			log.Warnf("failed to parse PR number for fragment file %q from string %q: %v", fragFile.Name(), commitMsg, err)
		}

		if prNum != 0 {
			for i, e := range fragment.Entries {
				if e.PullRequest == nil {
					fragment.Entries[i].PullRequest = &prNum
				}
			}
		}

		entries = append(entries, fragment.Entries...)
	}
	return entries, nil
}

func updateChangelog(c config) error {
	changelog := map[entryKind][]string{}
	for _, e := range c.Entries {
		changelog[e.Kind] = append(changelog[e.Kind], e.toChangelogString())
	}

	var bb bytes.Buffer
	order := []entryKind{
		addition,
		change,
		removal,
		deprecation,
		bugfix,
	}
	bb.WriteString(fmt.Sprintf("## v%s\n\n", c.Version))
	for _, k := range order {
		if entries, ok := changelog[k]; ok {
			bb.WriteString(k.toHeader() + "\n\n")
			for _, e := range entries {
				bb.WriteString(e + "\n")
			}
			bb.WriteString("\n")
		}
	}

	existingFile, err := ioutil.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("could not read CHANGELOG: %v", err)
	}
	bb.Write(existingFile)

	if err := ioutil.WriteFile(c.File, bb.Bytes(), 0644); err != nil {
		return fmt.Errorf("could not write CHANGELOG file: %v", err)
	}
	return nil
}

func createMigrationGuide(c config) error {
	var bb bytes.Buffer

	bb.WriteString("---\n")
	bb.WriteString(fmt.Sprintf("title: v%s\n", c.Version))
	bb.WriteString(fmt.Sprintf("weight: %d\n", convertVersionToWeight(c.Version)))
	bb.WriteString("---\n\n")
	haveMigrations := false
	for _, e := range c.Entries {
		if e.Migration != nil {
			haveMigrations = true
			bb.WriteString(fmt.Sprintf("## %s\n\n", e.Migration.Header))
			bb.WriteString(fmt.Sprintf("%s\n\n", strings.Trim(e.Migration.Body, "\n")))
			if e.PullRequest != nil {
				bb.WriteString(fmt.Sprintf("_See %s for more details._\n\n", e.pullRequestLink()))
			}
		}
	}
	if !haveMigrations {
		bb.WriteString("There are no migrations for this release! :tada:\n\n")
	}

	if err := ioutil.WriteFile(c.File, bytes.TrimSuffix(bb.Bytes(), []byte("\n")), 0644); err != nil {
		return fmt.Errorf("could not write migration guide: %v", err)
	}
	return nil
}

func convertVersionToWeight(v semver.Version) uint64 {
	return 1_000_000_000 - (v.Major * 1_000_000) - (v.Minor * 1_000) - v.Patch
}

type fragment struct {
	Entries []entry `yaml:"entries"`
}

type entry struct {
	Description string     `yaml:"description"`
	Kind        entryKind  `yaml:"kind"`
	Breaking    bool       `yaml:"breaking"`
	Migration   *migration `yaml:"migration,omitempty"`
	PullRequest *uint      `yaml:"pull_request_override,omitempty"`
}

type entryKind string

const (
	addition    entryKind = "addition"
	change      entryKind = "change"
	removal     entryKind = "removal"
	deprecation entryKind = "deprecation"
	bugfix      entryKind = "bugfix"
)

func (k entryKind) toHeader() string {
	switch k {
	case addition:
		return "### Additions"
	case change:
		return "### Changes"
	case removal:
		return "### Removals"
	case deprecation:
		return "### Deprecations"
	case bugfix:
		return "### Bug Fixes"
	default:
		panic("invalid entry kind; NOTE TO DEVELOPERS: check entryKind.validate")
	}
}

type migration struct {
	Header string `yaml:"header"`
	Body   string `yaml:"body"`
}

type config struct {
	File    string
	Version semver.Version
	Entries []entry
}

func (f *fragment) validate() error {
	for i, e := range f.Entries {
		if err := e.validate(); err != nil {
			return fmt.Errorf("entry[%d] invalid: %v", i, err)
		}
	}
	return nil
}

func (e *entry) validate() error {
	if err := e.Kind.validate(); err != nil {
		return fmt.Errorf("invalid kind: %v", err)
	}

	if e.Breaking && e.Kind != change && e.Kind != removal {
		return fmt.Errorf("breaking changes can only be kind %q or %q, got %q", change, removal, e.Kind)
	}

	if e.Breaking && e.Migration == nil {
		return fmt.Errorf("breaking changes require migration descriptions")
	}

	if e.Migration != nil {
		if err := e.Migration.validate(); err != nil {
			return fmt.Errorf("invalid migration: %v", err)
		}
	}
	return nil
}

func (e entry) toChangelogString() string {
	text := strings.TrimSpace(e.Description)
	if e.Breaking {
		text = fmt.Sprintf("**Breaking change**: %s", text)
	}
	if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") {
		text = fmt.Sprintf("%s.", text)
	}
	if e.PullRequest != nil {
		text = fmt.Sprintf("%s (%s)", text, e.pullRequestLink())
	}
	return fmt.Sprintf("- %s", text)
}

func (e entry) pullRequestLink() string {
	return fmt.Sprintf("[#%d](https://%s/pull/%d)", *e.PullRequest, repo, *e.PullRequest)
}

func getCommitMessage(filename string) (string, error) {
	args := fmt.Sprintf("log --follow --pretty=format:%%s --diff-filter=A --find-renames=40%% %s", filename)
	line, err := exec.Command("git", strings.Split(args, " ")...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to locate git commit for PR discovery: %v", err)
	}
	return string(line), nil
}

func parsePRNumber(msg string) (uint, error) {
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

func (k entryKind) validate() error {
	for _, t := range []entryKind{addition, change, removal, deprecation, bugfix} {
		if k == t {
			return nil
		}
	}
	return fmt.Errorf("%q is not a supported kind", k)
}

func (m migration) validate() error {
	if len(m.Header) == 0 {
		return errors.New("header not specified")
	}
	if len(m.Body) == 0 {
		return errors.New("body not specified")
	}
	return nil
}
