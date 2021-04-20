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

package pkgmantobundle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migrating packagemanifests to bundle command", func() {
	Describe("NewCmd", func() {
		cmd := NewCmd()
		Expect(cmd).NotTo(BeNil())

		flag := cmd.Flags().Lookup("output-dir")
		Expect(flag).NotTo(BeNil())
		Expect(flag.Usage).ToNot(Equal(""))
		Expect(flag.DefValue).To(Equal("bundles"))

		flag = cmd.Flags().Lookup("image-tag-base")
		Expect(flag).NotTo(BeNil())
		Expect(flag.Usage).ToNot(Equal(""))

		flag = cmd.Flags().Lookup("build-cmd")
		Expect(flag).NotTo(BeNil())
		Expect(flag.Usage).ToNot(Equal(""))
	})
})
