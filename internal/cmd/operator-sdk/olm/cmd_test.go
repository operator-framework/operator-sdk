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

package olm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running an olm command", func() {
	Describe("NewCmd", func() {
		It("builds a cobra command with the correct subcommands", func() {
			cmd := NewCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(BeNil())
			Expect(cmd.Short).NotTo(BeNil())

			subcommands := cmd.Commands()
			Expect(subcommands).To(HaveLen(3))
			Expect(subcommands[0].Use).To(Equal("install"))
			Expect(subcommands[1].Use).To(Equal("status"))
			Expect(subcommands[2].Use).To(Equal("uninstall"))
		})
	})
})
