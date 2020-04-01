package main

import (
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
	)

	flag.StringVar(&title, "title", "", "Title for generated CHANGELOG and migration guide sections")
	flag.StringVar(&fragmentsDir, "fragments-dir", filepath.Join("changelog", "fragments"), "Path to changelog fragments directory")
	flag.StringVar(&changelogFile, "changelog", "CHANGELOG.md", "Path to CHANGELOG.md")
	flag.Parse()

	if title == "" {
		log.Fatalf("flag '-title' is required!")
	}
	files, err := ioutil.ReadDir(fragmentsDir)
	if err != nil {
		log.Fatalf("failed to read fragments directory: %v", err)
	}

	changelog := map[EntryKind][]string{}
	haveEntries := false

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
			log.Fatalf("Failed to open fragment file %q: %v", fragFile.Name(), err)
		}

		decoder := yaml.NewDecoder(f)
		fragment := &Fragment{}
		if err := decoder.Decode(fragment); err != nil {
			log.Fatalf("Failed to parse fragment file %q: %v", fragFile.Name(), err)
		}

		if err := fragment.Validate(); err != nil {
			log.Fatalf("Failed to validate fragment file %q: %v", fragFile.Name(), err)
		}

		for _, e := range fragment.Entries {
			changelog[e.Kind] = append(changelog[e.Kind], e.ToChangelogString())
			haveEntries = true
		}
	}

	if !haveEntries {
		log.Fatal("No new CHANGELOG entries found!")
	}

	var sb strings.Builder
	order := []EntryKind{
		Addition,
		Change,
		Removal,
		Deprecation,
		Bugfix,
	}
	sb.WriteString(fmt.Sprintf("## %s\n\n", title))
	for _, k := range order {
		if entries, ok := changelog[k]; ok {
			sb.WriteString(k.ToHeader() + "\n\n")
			for _, e := range entries {
				sb.WriteString(e + "\n")
			}
			sb.WriteString("\n")
		}
	}

	existingFile, err := ioutil.ReadFile(changelogFile)
	if err != nil {
		log.Infof("No existing CHANGELOG file to prepend to")
	}
	sb.Write(existingFile)
	fmt.Print(sb.String())
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
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
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
		text = fmt.Sprintf("%s ([#%d](https://github.com/operator-framework/operator-sdk/pull/%d))", text, *e.PullRequest, *e.PullRequest)
	}
	return fmt.Sprintf("- %s", text)
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
	if len(m.Title) == 0 {
		return errors.New("title not specified")
	}
	if len(m.Body) == 0 {
		return errors.New("body not specified")
	}
	return nil
}
