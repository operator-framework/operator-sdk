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

package genutil_test

import (
	"fmt"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
)

var _ = Describe("ParseVersion", func() {
	vs := []string{
		"0.1.2-12-g453fb31",
		"v0.1.2-12-g453fb31",
	}
	for _, v := range vs {
		v := v
		It(fmt.Sprintf("should see %#v as valid semantic version", v), func() {
			sv, err := genutil.ParseVersion(v)
			Expect(err).To(BeNil())
			Expect(sv.Major).To(BeEquivalentTo(0))
			Expect(sv.Minor).To(BeEquivalentTo(1))
			Expect(sv.Patch).To(BeEquivalentTo(2))
			Expect(sv.Build).To(BeNil())
			Expect(sv.Pre).To(Equal([]semver.PRVersion{{
				VersionStr: "12-g453fb31",
			}}))
		})
	}
})
