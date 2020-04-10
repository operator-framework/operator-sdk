package util

import (
	"bytes"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/blang/semver"
)

type MigrationGuide struct {
	Version    string
	Weight     uint64
	Migrations []Migration
}

type Migration struct {
	Header          string
	Body            string
	PullRequestLink string
}

const migrationGuideTemplate = `---
title: v{{ .Version }}
weight: {{ .Weight }}
---
{{- range .Migrations }}
## {{ .Header }}

{{ .Body }}

_See {{ .PullRequestLink }} for more details._
{{ else }}
There are no migrations for this release! :tada:
{{- end }}
`

var migrationGuideTmpl = template.Must(template.New("migrationGuide").Parse(migrationGuideTemplate))

func (mg *MigrationGuide) Template() ([]byte, error) {
	w := &bytes.Buffer{}
	if err := migrationGuideTmpl.Execute(w, mg); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (mg *MigrationGuide) WriteFile(path string) error {
	data, err := mg.Template()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

func MigrationGuideFromEntries(version semver.Version, entries []FragmentEntry) MigrationGuide {
	mg := MigrationGuide{
		Version: version.String(),
		Weight:  versionToWeight(version),
	}
	for _, e := range entries {
		if e.Migration == nil {
			continue
		}
		mg.Migrations = append(mg.Migrations, Migration{
			Header:          e.Migration.Header,
			Body:            strings.TrimSpace(e.Migration.Body),
			PullRequestLink: e.PullRequestLink,
		})
	}
	return mg
}

func versionToWeight(v semver.Version) uint64 {
	return 1_000_000_000 - (v.Major * 1_000_000) - (v.Minor * 1_000) - v.Patch
}
