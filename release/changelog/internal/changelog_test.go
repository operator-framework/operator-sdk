package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
)

func getChangelogEntries(n int) []ChangelogEntry {
	entries := make([]ChangelogEntry, n)
	for i := 0; i < n; i++ {
		entries[i] = ChangelogEntry{
			Description: fmt.Sprintf("Changelog entry description %d.", i),
			Link:        "[#999999](https://example.com/test/changelog/pulls/999999)",
		}
	}
	return entries
}

func TestChangelog_Template(t *testing.T) {
	testCases := []struct {
		name      string
		changelog Changelog
		output    string
	}{
		{
			name: "all with 1 entry",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(1),
				Changes:      getChangelogEntries(1),
				Removals:     getChangelogEntries(1),
				Deprecations: getChangelogEntries(1),
				Bugfixes:     getChangelogEntries(1),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "all with 2 entries",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(2),
				Changes:      getChangelogEntries(2),
				Removals:     getChangelogEntries(2),
				Deprecations: getChangelogEntries(2),
				Bugfixes:     getChangelogEntries(2),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no additions",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    nil,
				Changes:      getChangelogEntries(1),
				Removals:     getChangelogEntries(1),
				Deprecations: getChangelogEntries(1),
				Bugfixes:     getChangelogEntries(1),
			},
			output: `## v999.999.999

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no changes",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(1),
				Changes:      nil,
				Removals:     getChangelogEntries(1),
				Deprecations: getChangelogEntries(1),
				Bugfixes:     getChangelogEntries(1),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no removals",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(1),
				Changes:      getChangelogEntries(1),
				Removals:     nil,
				Deprecations: getChangelogEntries(1),
				Bugfixes:     getChangelogEntries(1),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no deprecations",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(1),
				Changes:      getChangelogEntries(1),
				Removals:     getChangelogEntries(1),
				Deprecations: nil,
				Bugfixes:     getChangelogEntries(1),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no bug fixes",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(1),
				Changes:      getChangelogEntries(1),
				Removals:     getChangelogEntries(1),
				Deprecations: getChangelogEntries(1),
				Bugfixes:     nil,
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "entry with no link",
			changelog: Changelog{
				Version: "v999.999.999",
				Additions: []ChangelogEntry{
					{
						Description: "Changelog entry description 0.",
					},
					{
						Description: "Changelog entry description 1.",
						Link:        "[#999999](https://example.com/test/changelog/pulls/999999)",
					},
				},
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0.
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "no entries",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    nil,
				Changes:      nil,
				Removals:     nil,
				Deprecations: nil,
				Bugfixes:     nil,
			},
			output: `## v999.999.999

No changes for this release!
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := tc.changelog.Template()
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
			assert.Equal(t, tc.output, string(d))
		})
	}
}

func TestChangelog_WriteFile(t *testing.T) {

	testCases := []struct {
		name                 string
		changelog            Changelog
		existingFile         bool
		existingFileContents string
		output               string
	}{
		{
			name: "non-existent file",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(2),
				Changes:      getChangelogEntries(2),
				Removals:     getChangelogEntries(2),
				Deprecations: getChangelogEntries(2),
				Bugfixes:     getChangelogEntries(2),
			},
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "empty file",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(2),
				Changes:      getChangelogEntries(2),
				Removals:     getChangelogEntries(2),
				Deprecations: getChangelogEntries(2),
				Bugfixes:     getChangelogEntries(2),
			},
			existingFile: true,
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
		{
			name: "existing file",
			changelog: Changelog{
				Version:      "v999.999.999",
				Additions:    getChangelogEntries(2),
				Changes:      getChangelogEntries(2),
				Removals:     getChangelogEntries(2),
				Deprecations: getChangelogEntries(2),
				Bugfixes:     getChangelogEntries(2),
			},
			existingFileContents: `## v999.999.998

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
			output: `## v999.999.999

### Additions

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Changes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Removals

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Deprecations

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))

## v999.999.998

### Bug Fixes

- Changelog entry description 0. ([#999999](https://example.com/test/changelog/pulls/999999))
- Changelog entry description 1. ([#999999](https://example.com/test/changelog/pulls/999999))
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile("", "go-test-changelog")
			assert.NoError(t, err)
			assert.NoError(t, tmpFile.Close())
			defer assert.NoError(t, os.Remove(tmpFile.Name()))

			if tc.existingFile || len(tc.existingFileContents) > 0 {
				assert.NoError(t, ioutil.WriteFile(tmpFile.Name(), []byte(tc.existingFileContents), 0644))
			}

			assert.NoError(t, tc.changelog.WriteFile(tmpFile.Name()))

			d, err := ioutil.ReadFile(tmpFile.Name())
			assert.NoError(t, err)
			assert.Equal(t, tc.output, string(d))
		})
	}
}

func TestChangelog_ChangelogFromEntries(t *testing.T) {
	testCases := []struct {
		name      string
		version   semver.Version
		entries   []FragmentEntry
		changelog Changelog
	}{
		{
			name:      "no entries",
			version:   semver.MustParse("999.999.999"),
			changelog: Changelog{Version: "v999.999.999"},
		},
		{
			name:    "add periods to descriptions and breaking change prefix",
			version: semver.MustParse("999.999.999"),
			entries: []FragmentEntry{
				{
					Description:     "Changelog entry description 0",
					Kind:            Addition,
					Breaking:        false,
					PullRequestLink: "[#999999](https://example.com/test/changelog/pulls/999999)",
				},
				{
					Description: "Changelog entry description 0",
					Kind:        Change,
					Breaking:    true,
				},
				{
					Description: "Changelog entry description 0",
					Kind:        Removal,
					Breaking:    true,
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
			changelog: Changelog{
				Version: "v999.999.999",
				Additions: []ChangelogEntry{
					{
						Description: "Changelog entry description 0.",
						Link:        "[#999999](https://example.com/test/changelog/pulls/999999)",
					},
				},
				Changes: []ChangelogEntry{
					{
						Description: "**Breaking change**: Changelog entry description 0.",
					},
				},
				Removals: []ChangelogEntry{
					{
						Description: "**Breaking change**: Changelog entry description 0.",
					},
				},
				Deprecations: []ChangelogEntry{
					{
						Description: "Changelog entry description 0.",
					},
				},
				Bugfixes: []ChangelogEntry{
					{
						Description: "Changelog entry description 0.",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := ChangelogFromEntries(tc.version, tc.entries)
			assert.Equal(t, tc.changelog, cl)
		})
	}
}
