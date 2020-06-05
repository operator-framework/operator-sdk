package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockValidPRGetter struct{}

var _ PullRequestNumberGetter = &mockValidPRGetter{}

func (m *mockValidPRGetter) GetPullRequestNumberFor(file string) (uint, error) {
	return 999998, nil
}

func TestFragment_LoadEntries(t *testing.T) {
	discoveredPRNum := uint(999998)
	overriddenPRNum := uint(999999)
	repoLink := "example.com/test/changelog"

	testCases := []struct {
		name            string
		fragmentsDir    string
		prGetter        PullRequestNumberGetter
		expectedEntries []FragmentEntry
		expectedErr     string
	}{
		{
			name:            "ignore non-fragments",
			fragmentsDir:    "testdata/ignore",
			expectedEntries: nil,
		},
		{
			name:            "invalid yaml",
			fragmentsDir:    "testdata/invalid_yaml",
			expectedEntries: nil,
			expectedErr:     "error unmarshaling",
		},
		{
			name:            "invalid entry",
			fragmentsDir:    "testdata/invalid_entry",
			expectedEntries: nil,
			expectedErr:     `failed to validate fragment file`,
		},
		{
			name:         "valid fragments",
			fragmentsDir: "testdata/valid",
			prGetter:     &mockValidPRGetter{},
			expectedEntries: []FragmentEntry{
				{
					Description:     "Addition description 0",
					Kind:            Addition,
					PullRequest:     &discoveredPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", discoveredPRNum, repoLink, discoveredPRNum),
				},
				{
					Description:     "Change description 0",
					Kind:            Change,
					PullRequest:     &discoveredPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", discoveredPRNum, repoLink, discoveredPRNum),
				},
				{
					Description: "Removal description 0",
					Kind:        Removal,
					Breaking:    true,
					Migration: &EntryMigration{
						Header: "Header for removal migration 0",
						Body:   "Body for removal migration 0",
					},
					PullRequest:     &discoveredPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", discoveredPRNum, repoLink, discoveredPRNum),
				},
				{
					Description:     "Deprecation description 0",
					Kind:            Deprecation,
					PullRequest:     &discoveredPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", discoveredPRNum, repoLink, discoveredPRNum),
				},
				{
					Description:     "Bugfix description 0",
					Kind:            Bugfix,
					PullRequest:     &discoveredPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", discoveredPRNum, repoLink, discoveredPRNum),
				},
				{
					Description:     "Addition description 1",
					Kind:            Addition,
					PullRequest:     &overriddenPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", overriddenPRNum, repoLink, overriddenPRNum),
				},
				{
					Description:     "Change description 1",
					Kind:            Change,
					PullRequest:     &overriddenPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", overriddenPRNum, repoLink, overriddenPRNum),
				},
				{
					Description: "Removal description 1",
					Kind:        Removal,
					Breaking:    true,
					Migration: &EntryMigration{
						Header: "Header for removal migration 1",
						Body:   "Body for removal migration 1",
					},
					PullRequest:     &overriddenPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", overriddenPRNum, repoLink, overriddenPRNum),
				},
				{
					Description:     "Deprecation description 1",
					Kind:            Deprecation,
					PullRequest:     &overriddenPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", overriddenPRNum, repoLink, overriddenPRNum),
				},
				{
					Description:     "Bugfix description 1",
					Kind:            Bugfix,
					PullRequest:     &overriddenPRNum,
					PullRequestLink: fmt.Sprintf("[#%d](https://%s/pull/%d)", overriddenPRNum, repoLink, overriddenPRNum),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prGetter = tc.prGetter
			entries, err := LoadEntries(tc.fragmentsDir, repoLink)
			assert.Equal(t, tc.expectedEntries, entries)
			if len(tc.expectedErr) > 0 {
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("expected error to contain: %q, got %q", tc.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFragmentEntry_Validate(t *testing.T) {
	testCases := []struct {
		name          string
		fragmentEntry FragmentEntry
		expectedErr   string
	}{
		{
			name: "invalid kind",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        "invalid",
				Breaking:    false,
			},
			expectedErr: "invalid kind",
		},
		{
			name: "missing description",
			fragmentEntry: FragmentEntry{
				Description: "",
				Kind:        Addition,
				Breaking:    false,
			},
			expectedErr: "missing description",
		},
		{
			name: "breaking addition not allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Addition,
				Breaking:    true,
			},
			expectedErr: `breaking changes can only be kind "change" or "removal", got "addition"`,
		},
		{
			name: "breaking deprecation not allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Deprecation,
				Breaking:    true,
			},
			expectedErr: `breaking changes can only be kind "change" or "removal", got "deprecation"`,
		},
		{
			name: "breaking bugfix not allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Bugfix,
				Breaking:    true,
			},
			expectedErr: `breaking changes can only be kind "change" or "removal", got "bugfix"`,
		},
		{
			name: "migration missing",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Change,
				Breaking:    true,
			},
			expectedErr: `breaking changes require migration sections`,
		},
		{
			name: "migration header missing",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Change,
				Breaking:    true,
				Migration: &EntryMigration{
					Header: "",
					Body:   "migration body",
				},
			},
			expectedErr: `invalid migration: header not specified`,
		},
		{
			name: "migration body missing",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Change,
				Breaking:    true,
				Migration: &EntryMigration{
					Header: "migration header",
					Body:   "",
				},
			},
			expectedErr: `invalid migration: body not specified`,
		},
		{
			name: "breaking change allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Change,
				Breaking:    true,
				Migration: &EntryMigration{
					Header: "migration header",
					Body:   "migration body",
				},
			},
			expectedErr: ``,
		},
		{
			name: "breaking removal allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Removal,
				Breaking:    true,
				Migration: &EntryMigration{
					Header: "migration header",
					Body:   "migration body",
				},
			},
			expectedErr: ``,
		},
		{
			name: "non-breaking migration allowed",
			fragmentEntry: FragmentEntry{
				Description: "description",
				Kind:        Addition,
				Breaking:    false,
				Migration: &EntryMigration{
					Header: "migration header",
					Body:   "migration body",
				},
			},
			expectedErr: ``,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fragmentEntry.Validate()

			if len(tc.expectedErr) == 0 {
				assert.NoError(t, err)
			} else if err == nil {
				t.Errorf("expected error to contain %q, got no error", tc.expectedErr)
			} else {
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("expected error to contain: %q, got %q", tc.expectedErr, err)
				}
			}
		})
	}
}

func TestFragmentEntry_PullRequestLink(t *testing.T) {
	prNum := uint(999999)
	testCases := []struct {
		name          string
		fragmentEntry FragmentEntry
		repo          string
		link          string
	}{
		{
			name:          "no link",
			fragmentEntry: FragmentEntry{},
			link:          "",
		},
		{
			name:          "link with repo 1",
			repo:          "example.com/test/repo1",
			fragmentEntry: FragmentEntry{PullRequest: &prNum},
			link:          "[#999999](https://example.com/test/repo1/pull/999999)",
		},
		{
			name:          "link with repo 2",
			repo:          "example.com/test/repo2",
			fragmentEntry: FragmentEntry{PullRequest: &prNum},
			link:          "[#999999](https://example.com/test/repo2/pull/999999)",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			link := tc.fragmentEntry.pullRequestLink(tc.repo)
			assert.Equal(t, tc.link, link)
		})
	}
}

func TestGitPullRequestNumberGetter_parsePRNumber(t *testing.T) {
	prg := &gitPullRequestNumberGetter{}
	testCases := []struct {
		name        string
		msg         string
		prNum       uint
		expectedErr string
	}{
		{
			name:  "valid message",
			msg:   "this is a message with a PR number at the end (#999999)",
			prNum: uint(999999),
		},
		{
			name:        "missing parentheses",
			msg:         "this is a message with a PR number at the end #999999",
			expectedErr: `could not find PR number in commit message`,
		},
		{
			name:        "not at the end",
			msg:         "this is a message with a PR number (#999999) in the middle",
			expectedErr: `could not find PR number in commit message`,
		},
		{
			name:        "no PR number",
			msg:         "this is a message without a PR number",
			expectedErr: `could not find PR number in commit message`,
		},
		{
			name:        "invalid PR number",
			msg:         "this is a message with a really big PR number (#99999999999999999999999999999999999999)",
			expectedErr: `value out of range`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prNum, err := prg.parsePRNumber(tc.msg)

			if len(tc.expectedErr) == 0 {
				assert.NoError(t, err)
			} else if err == nil {
				t.Errorf("expected error to contain %q, got no error", tc.expectedErr)
			} else {
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("expected error to contain: %q, got %q", tc.expectedErr, err)
				}
			}

			assert.Equal(t, tc.prNum, prNum)
		})
	}
}

func Test_gitPullRequestNumberGetter_getCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "should fail when the fragment file cannot be found in any commit",
			filename: "/does/not/exist",
			wantErr:  true,
		},
		{
			name:     "should work successfully when there is a commit with the fragment file",
			filename: "testdata/valid/fragment1.yaml",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			g := &gitPullRequestNumberGetter{}
			_, err := g.getCommitMessage(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
