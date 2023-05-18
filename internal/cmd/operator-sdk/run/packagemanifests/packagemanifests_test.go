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

package packagemanifests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("Running a run packagemanifests command", func() {
	Describe("NewCmd", func() {
		It("builds a cobra command", func() {
			cfg := &operator.Configuration{}
			cmd := NewCmd(cfg)
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(BeNil())
			Expect(cmd.Short).NotTo(BeNil())
			Expect(cmd.Long).NotTo(BeNil())
			aliases := cmd.Aliases
			Expect(aliases).To(HaveLen(1))
			Expect(aliases[0]).To(Equal("pm"))
		})
	})
})
