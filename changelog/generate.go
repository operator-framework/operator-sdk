package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		title         string
		fragmentsDir  string
		changelogFile string
		migrationFile string
	)

	flag.StringVar(&title, "title", "",
		"Title for generated CHANGELOG and migration guide sections")
	flag.StringVar(&fragmentsDir, "fragments-dir", filepath.Join("changelog", "fragments"),
		"Path to changelog fragments directory")
	flag.StringVar(&changelogFile, "changelog", "CHANGELOG.md",
		"Path to CHANGELOG")
	flag.StringVar(&migrationFile, "migration-guide",
		filepath.Join("website", "content", "en", "docs", "migration", "version-upgrade-guide.md"),
		"Path to migration guide")
	flag.Parse()

	if title == "" {
		log.Fatalf("flag '-title' is required!")
	}

	entries, err := LoadEntries(fragmentsDir)
	if err != nil {
		log.Fatalf("failed to load fragments: %v", err)
	}
	if len(entries) == 0 {
		log.Fatalf("no entries found")
	}

	if err := UpdateChangelog(Config{
		File:    changelogFile,
		Title:   title,
		Entries: entries,
	}); err != nil {
		log.Fatalf("failed to update CHANGELOG: %v", err)
	}

	if err := UpdateMigrationGuide(Config{
		File:    migrationFile,
		Title:   title,
		Entries: entries,
	}); err != nil {
		log.Fatalf("failed to update migration guide: %v", err)
	}
}

func LoadEntries(fragmentsDir string) ([]Entry, error) {
	files, err := ioutil.ReadDir(fragmentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read fragments directory: %w", err)
	}

	var entries []Entry
	for _, fragFile := range files {
		if fragFile.Name() == "00-template.yaml" {
			continue
		}
		if fragFile.IsDir() {
			log.Warnf("Skipping directory %q", fragFile.Name())
			continue
		}
		if filepath.Ext(fragFile.Name()) != ".yaml" || fragFile.IsDir() {
			log.Warnf("Skipping non-YAML file %q", fragFile.Name())
			continue
		}
		path := filepath.Join(fragmentsDir, fragFile.Name())
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open fragment file %q: %w", fragFile.Name(), err)
		}

		decoder := yaml.NewDecoder(f)
		fragment := Fragment{}
		if err := decoder.Decode(&fragment); err != nil {
			return nil, fmt.Errorf("failed to parse fragment file %q: %w", fragFile.Name(), err)
		}

		if err := fragment.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate fragment file %q: %w", fragFile.Name(), err)
		}

		entries = append(entries, fragment.Entries...)
	}
	return entries, nil
}

func UpdateChangelog(c Config) error {
	changelog := map[EntryKind][]string{}
	for _, e := range c.Entries {
		changelog[e.Kind] = append(changelog[e.Kind], e.ToChangelogString())
	}

	var bb bytes.Buffer
	order := []EntryKind{
		Addition,
		Change,
		Removal,
		Deprecation,
		Bugfix,
	}
	bb.WriteString(fmt.Sprintf("## %s\n\n", c.Title))
	for _, k := range order {
		if entries, ok := changelog[k]; ok {
			bb.WriteString(k.ToHeader() + "\n\n")
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

func UpdateMigrationGuide(c Config) error {
	var bb bytes.Buffer
	existingFile, err := ioutil.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("could not read migration guide: %v", err)
	}
	bb.Write(bytes.Trim(existingFile, "\n"))

	bb.WriteString(fmt.Sprintf("\n\n## %s\n\n", c.Title))
	haveMigrations := false
	for _, e := range c.Entries {
		if e.Migration != nil {
			haveMigrations = true
			bb.WriteString(fmt.Sprintf("### %s\n\n", e.Migration.Header))
			bb.WriteString(fmt.Sprintf("%s\n\n", strings.Trim(e.Migration.Body, "\n")))
			bb.WriteString(fmt.Sprintf("See %s for more details.\n\n", e.PullRequestLink()))
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

type Fragment struct {
	Entries []Entry `yaml:"entries"`
}

type Entry struct {
	Description string     `yaml:"description"`
	Kind        EntryKind  `yaml:"kind"`
	Breaking    bool       `yaml:"breaking"`
	Migration   *Migration `yaml:"migration,omitempty"`
	PullRequest *uint      `yaml:"pull_request,omitempty"`
}

type EntryKind string

const (
	Addition    EntryKind = "addition"
	Change      EntryKind = "change"
	Removal     EntryKind = "removal"
	Deprecation EntryKind = "deprecation"
	Bugfix      EntryKind = "bugfix"
)

func (k EntryKind) ToHeader() string {
	switch k {
	case Addition:
		return "### Additions"
	case Change:
		return "### Changes"
	case Removal:
		return "### Removals"
	case Deprecation:
		return "### Deprecations"
	case Bugfix:
		return "### Bug Fixes"
	default:
		panic("invalid entry kind; NOTE TO DEVELOPERS: check EntryKind.Validate")
	}
}

type Migration struct {
	Header string `yaml:"header"`
	Body   string `yaml:"body"`
}

type Config struct {
	File    string
	Title   string
	Entries []Entry
}

func (f *Fragment) Validate() error {
	for i, e := range f.Entries {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("entry[%d] invalid: %v", i, err)
		}
	}
	return nil
}

func (e *Entry) Validate() error {
	if err := e.Kind.Validate(); err != nil {
		return fmt.Errorf("invalid kind: %v", err)
	}

	if e.Breaking && e.Kind != Change && e.Kind != Removal {
		return fmt.Errorf("breaking changes can only be kind %q or %q, got %q", Change, Removal, e.Kind)
	}

	if e.Breaking && e.Migration == nil {
		return fmt.Errorf("breaking changes require migration descriptions")
	}

	if e.Migration != nil {
		if err := e.Migration.Validate(); err != nil {
			return fmt.Errorf("invalid migration: %v", err)
		}
	}
	return nil
}

func (e Entry) ToChangelogString() string {
	text := strings.Trim(e.Description, "\n")
	if e.Breaking {
		text = fmt.Sprintf("**Breaking Change**: %s", text)
	}
	if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") {
		text = fmt.Sprintf("%s.", text)
	}
	if e.PullRequest != nil {
		text = fmt.Sprintf("%s (%s)", text, e.PullRequestLink())
	}
	return fmt.Sprintf("- %s", text)
}

func (e Entry) PullRequestLink() string {
	const repo = "github.com/operator-framework/operator-sdk"
	return fmt.Sprintf("[#%d](https://%s/pull/%d)", *e.PullRequest, repo, *e.PullRequest)
}

func (k EntryKind) Validate() error {
	for _, t := range []EntryKind{Addition, Change, Removal, Deprecation, Bugfix} {
		if k == t {
			return nil
		}
	}
	return fmt.Errorf("%q is not a supported kind", k)
}

func (m Migration) Validate() error {
	if len(m.Header) == 0 {
		return errors.New("header not specified")
	}
	if len(m.Body) == 0 {
		return errors.New("body not specified")
	}
	return nil
}
