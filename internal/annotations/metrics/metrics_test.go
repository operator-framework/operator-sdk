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

package metrics

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SDK Label helper functions", func() {
	Describe("parseVersion", func() {
		It("should extract sdk version", func() {
			version := "v0.17.0-159-ge87627f4-dirty"
			output := parseVersion(version)
			Expect(output).To(Equal("v0.17.0+git"))
		})
		It("should extract sdk version", func() {
			version := "v0.18.0"
			output := parseVersion(version)
			Expect(output).To(Equal("v0.18.0"))
		})
		It("should extract sdk version", func() {
			version := "v0.18.0-ge87627f4"
			output := parseVersion(version)
			Expect(output).To(Equal("v0.18.0+git"))
		})

	})
})
