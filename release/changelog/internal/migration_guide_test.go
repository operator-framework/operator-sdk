package util

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
)

func TestMigrationGuide_Template(t *testing.T) {
	testCases := []struct {
		name   string
		mg     MigrationGuide
		output string
	}{
		{
			name: "link_then_no_link",
			mg: MigrationGuide{
				Version: "v999.999.999",
				Weight:  1,
				Migrations: []Migration{
					{
						Header:          "Migration header 0",
						Body:            "Migration body 0",
						PullRequestLink: "[#999999](https://example.com/test/changelog/pull/999999)",
					},
					{
						Header: "Migration header 1",
						Body:   "Migration body 1",
					},
				},
			},
			output: `---
title: v999.999.999
weight: 1
---

## Migration header 0

Migration body 0

_See [#999999](https://example.com/test/changelog/pull/999999) for more details._

## Migration header 1

Migration body 1
`,
		},
		{
			name: "no_link_then_link",
			mg: MigrationGuide{
				Version: "v999.999.999",
				Weight:  2,
				Migrations: []Migration{
					{
						Header: "Migration header 0",
						Body:   "Migration body 0",
					},
					{
						Header:          "Migration header 1",
						Body:            "Migration body 1",
						PullRequestLink: "[#999999](https://example.com/test/changelog/pull/999999)",
					},
				},
			},
			output: `---
title: v999.999.999
weight: 2
---

## Migration header 0

Migration body 0

## Migration header 1

Migration body 1

_See [#999999](https://example.com/test/changelog/pull/999999) for more details._
`,
		},
		{
			name: "no migrations",
			mg: MigrationGuide{
				Version: "v999.999.999",
				Weight:  3,
			},
			output: `---
title: v999.999.999
weight: 3
---

There are no migrations for this release! :tada:
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := tc.mg.Template()
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
			assert.Equal(t, tc.output, string(d))
		})
	}
}

func TestMigrationGuide_WriteFile(t *testing.T) {

	testCases := []struct {
		name   string
		mg     MigrationGuide
		output string
	}{
		{
			name: "valid",
			mg: MigrationGuide{
				Version: "v999.999.999",
				Migrations: []Migration{
					{
						Header:          "Migration header 0",
						Body:            "Migration body 0",
						PullRequestLink: "[#999999](https://example.com/test/changelog/pull/999999)",
					},
					{
						Header: "Migration header 1",
						Body:   "Migration body 1",
					},
				},
			},
			output: `---
title: v999.999.999
weight: 0
---

## Migration header 0

Migration body 0

_See [#999999](https://example.com/test/changelog/pull/999999) for more details._

## Migration header 1

Migration body 1
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile("", "go-test-changelog")
			assert.NoError(t, err)
			assert.NoError(t, tmpFile.Close())
			defer assert.NoError(t, os.Remove(tmpFile.Name()))

			assert.NoError(t, tc.mg.WriteFile(tmpFile.Name()))

			d, err := ioutil.ReadFile(tmpFile.Name())
			assert.NoError(t, err)
			assert.Equal(t, tc.output, string(d))
		})
	}
}

func TestMigrationGuide_MigrationGuideFromEntries(t *testing.T) {
	testCases := []struct {
		name    string
		version semver.Version
		entries []FragmentEntry
		mg      MigrationGuide
	}{
		{
			name:    "no entries, weight 1",
			version: semver.MustParse("999.999.999"),
			mg: MigrationGuide{
				Version: "v999.999.999",
				Weight:  1,
			},
		},
		{
			name:    "no entries, weight 2",
			version: semver.MustParse("999.999.998"),
			mg: MigrationGuide{
				Version: "v999.999.998",
				Weight:  2,
			},
		},
		{
			name:    "no entries, weight 998_997_997",
			version: semver.MustParse("1.2.3"),
			mg: MigrationGuide{
				Version: "v1.2.3",
				Weight:  998_997_997,
			},
		},
		{
			name:    "no entries, weight 2",
			version: semver.MustParse("3.2.1"),
			mg: MigrationGuide{
				Version: "v3.2.1",
				Weight:  996_997_999,
			},
		},
		{
			name:    "some migrations",
			version: semver.MustParse("999.999.999"),
			entries: []FragmentEntry{
				{
					Description:     "Changelog entry description 0",
					Kind:            Addition,
					Breaking:        false,
					PullRequestLink: "[#999999](https://example.com/test/changelog/pulls/999999)",
					Migration: &EntryMigration{
						Header: "Migration header 0",
						Body:   "Migration body 0",
					},
				},
				{
					Description: "Changelog entry description 1",
					Kind:        Change,
					Breaking:    true,
					Migration: &EntryMigration{
						Header: "Migration header 1",
						Body:   "Migration body 1",
					},
				},
				{
					Description: "Changelog entry description 2",
					Kind:        Removal,
					Breaking:    true,
					Migration: &EntryMigration{
						Header: "Migration header 2",
						Body:   "Migration body 2",
					},
				},
				{
					Description: "Changelog entry description 0",
					Kind:        Deprecation,
					Breaking:    false,
				},
				{
					Description: "Changelog entry description 0",
					Kind:        Bugfix,
					Breaking:    false,
				},
			},
			mg: MigrationGuide{
				Version: "v999.999.999",
				Weight:  1,
				Migrations: []Migration{
					{
						Header:          "Migration header 0",
						Body:            "Migration body 0",
						PullRequestLink: "[#999999](https://example.com/test/changelog/pulls/999999)",
					},
					{
						Header: "Migration header 1",
						Body:   "Migration body 1",
					},
					{
						Header: "Migration header 2",
						Body:   "Migration body 2",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mg := MigrationGuideFromEntries(tc.version, tc.entries)
			assert.Equal(t, tc.mg, mg)
		})
	}
}
