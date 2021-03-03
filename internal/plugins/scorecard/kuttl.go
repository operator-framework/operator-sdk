// Copyright 2021 The Operator-SDK Authors
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

package scorecard

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
)

var _ file.Inserter = &kuttlTestSuite{}

// kuttlTestSuite scaffolds or updates the kuttl-test.yaml in test/kuttl.
type kuttlTestSuite struct {
	file.InserterMixin

	relCaseDir string
}

const kuttlTestDirsMarker = "kuttl:testDirs"

// GetMarkers implements file.Inserter
func (f *kuttlTestSuite) GetMarkers() []file.Marker {
	return []file.Marker{file.NewMarkerFor(f.Path, kuttlTestDirsMarker)}
}

// GetCodeFragments implements file.Inserter
func (f *kuttlTestSuite) GetCodeFragments() file.CodeFragmentsMap {
	return file.CodeFragmentsMap{
		file.NewMarkerFor(f.Path, kuttlTestDirsMarker): []string{fmt.Sprintf("- %s\n", f.relCaseDir)},
	}
}
