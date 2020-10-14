// Copyright 2020 The Operator-SDK Authors
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

package pkg

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// SampleContext represents the Context used to generate the samples
type SampleContext struct {
	testutils.TestContext
}

// NewSampleContext returns a SampleContext containing a new kubebuilder TestContext.
func NewSampleContext(binary string, path string, env ...string) (s SampleContext, err error) {
	s.TestContext, err = testutils.NewTestContext(binary, env...)
	// If the path was informed then this should be the dir used
	if strings.TrimSpace(path) != "" {
		path, err = filepath.Abs(path)
		if err != nil {
			return s, err
		}
		s.CmdContext.Dir = path
		s.ProjectName = strings.ToLower(filepath.Base(s.Dir))
		s.ImageName = fmt.Sprintf("quay.io/example/%s:v0.0.1", s.ProjectName)
		s.BundleImageName = fmt.Sprintf("quay.io/example/%s-bundle:v0.0.1", s.ProjectName)
	}

	return s, err
}

// NewSampleContextWithTestContext returns a SampleContext containing the kubebuilder TestContext informed
// It is useful to allow the samples code be re-used in the e2e tests.
func NewSampleContextWithTestContext(tc *testutils.TestContext) (s SampleContext, err error) {
	s.TestContext = *tc
	return s, err
}
