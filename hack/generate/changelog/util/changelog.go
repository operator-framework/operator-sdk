package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/blang/semver"
)

type Changelog struct {
	Version      string
	Additions    []ChangelogEntry
	Changes      []ChangelogEntry
	Removals     []ChangelogEntry
	Deprecations []ChangelogEntry
	Bugfixes     []ChangelogEntry

	Repo string
}

type ChangelogEntry struct {
	Description string
	Link        string
}

const changelogTemplate = `## v{{ .Version }}
{{- with .Additions }}

### Additions
{{ range . }}
- {{ .Description }} ({{ .Link }})
{{- end }}{{- end }}
{{- with .Changes }}

### Changes
{{ range . }}
- {{ .Description }} ({{ .Link }})
{{- end }}{{- end }}
{{- with .Removals }}

### Removals
{{ range . }}
- {{ .Description }} ({{ .Link }})
{{- end }}{{- end }}
{{- with .Deprecations }}

### Deprecations
{{ range . }}
- {{ .Description }} ({{ .Link }})
{{- end }}{{- end }}
{{- with .Bugfixes }}

### Bug Fixes
{{ range . }}
- {{ .Description }} ({{ .Link }})
{{- end }}{{- end }}`

var changelogTmpl = template.Must(template.New("changelog").Parse(changelogTemplate))

func (c *Changelog) Template() ([]byte, error) {
	w := &bytes.Buffer{}
	if err := changelogTmpl.Execute(w, c); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (c *Changelog) WriteFile(path string) error {
	data, err := c.Template()
	if err != nil {
		return err
	}
	existingFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	data = append(data, '\n', '\n')
	data = append(data, existingFile...)
	return ioutil.WriteFile(path, data, 0644)
}

func ChangelogFromEntries(version semver.Version, entries []FragmentEntry) Changelog {
	cl := Changelog{
		Version: version.String(),
	}
	for _, e := range entries {
		cle := e.ToChangelogEntry()
		switch e.Kind {
		case Addition:
			cl.Additions = append(cl.Additions, cle)
		case Change:
			cl.Changes = append(cl.Changes, cle)
		case Removal:
			cl.Removals = append(cl.Removals, cle)
		case Deprecation:
			cl.Deprecations = append(cl.Deprecations, cle)
		case Bugfix:
			cl.Bugfixes = append(cl.Bugfixes, cle)
		}
	}
	return cl
}

func (e *FragmentEntry) ToChangelogEntry() ChangelogEntry {
	cle := ChangelogEntry{}
	desc := strings.TrimSpace(e.Description)
	if e.Breaking {
		desc = fmt.Sprintf("**Breaking change**: %s", desc)
	}
	if !strings.HasSuffix(desc, ".") && !strings.HasSuffix(desc, "!") {
		desc = fmt.Sprintf("%s.", desc)
	}
	cle.Description = desc
	cle.Link = e.PullRequestLink
	return cle
}
