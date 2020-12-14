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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/google/go-github/v33/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	path = fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
)

func setup(tagName string) (*github.Client, func()) {
	handler := http.NewServeMux()
	srv := httptest.NewServer(handler)

	serverURL, _ := url.Parse(srv.URL + "/")

	client := github.NewClient(srv.Client())
	client.BaseURL = serverURL

	handler.HandleFunc(path, func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "{\"tag_name\": \"%s\"}", tagName)
	})

	return client, func() { srv.Close() }
}

var _ = Describe("notification", func() {
	Describe("PrintUpdateNotification()", func() {
		It("prints update notification", func() {
			PrintUpdateNotification()
		})
	})

	Describe("getLatestVersionFromGithub()", func() {
		It("returns latest release", func() {
			tagName := "v100.0.0"

			client, teardownFunc := setup(tagName)
			defer teardownFunc()

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()

			latestVersion, err := getLatestVersionFromGithub(ctx, client, owner, repo)
			Expect(err).To(BeNil())

			expected, err := semver.Make(strings.TrimPrefix(tagName, "v"))
			Expect(err).To(BeNil())
			Expect(latestVersion).To(Equal(expected))
		})
		It("doesn't return latest release, context deadline exceeded", func() {
			tagName := "v100.0.0"

			client, teardownFunc := setup(tagName)
			defer teardownFunc()

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()

			time.Sleep(time.Second)
			latestVersion, err := getLatestVersionFromGithub(ctx, client, owner, repo)
			Expect(err).ToNot(BeNil())
			expectedError := context.DeadlineExceeded.Error()
			Expect(err.Error()).To(Equal(expectedError))
			Expect(latestVersion.String()).To(Equal("0.0.0"))
		})
		It("returns latest release, doesn't follow semantic versioning", func() {
			tagName := ""

			client, teardownFunc := setup(tagName)
			defer teardownFunc()

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
			defer cancel()

			latestVersion, err := getLatestVersionFromGithub(ctx, client, owner, repo)
			Expect(err).ToNot(BeNil())
			Expect(latestVersion.String()).To(Equal("0.0.0"))
		})
	})
})
