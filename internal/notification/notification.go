// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/google/go-github/v33/github"
	"github.com/operator-framework/operator-sdk/internal/version"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	owner = "operator-framework"
	repo  = "operator-sdk"
)

var log = logf.Log.WithName("notification")

// PrintUpdateNotification prints an update notification
// if update is available in ReleaseURL
func PrintUpdateNotification() {
	localVersion, err := semver.Make(strings.TrimPrefix(version.Version, "v"))
	if err != nil {
		log.Error(err, "Failed to parse local version")
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()
	client := github.NewClient(nil)
	latestVersion, err := getLatestVersionFromGithub(ctx, client, owner, repo)
	if err != nil {
		log.Error(err, "Failed to get latest version information")
	}
	// Example: https://github.com/operator-framework/operator-sdk/releases/tag/v1.2.0
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", owner, repo, latestVersion.String())
	if localVersion.Compare(latestVersion) < 0 {
		fmt.Printf("New version is available! Download it: %s\n\n", downloadURL)
	}
}

func getLatestVersionFromGithub(ctx context.Context, client *github.Client, owner, repo string) (semver.Version, error) {
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return semver.Version{}, err
	}

	latestVersion, err := semver.Make(strings.TrimPrefix(*release.TagName, "v"))
	if err != nil {
		return semver.Version{}, err
	}

	return latestVersion, nil
}
