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

	"github.com/operator-framework/operator-sdk/internal/olm/installer"
)

var _ = Describe("Running an olm install command", func() {
	Describe("newInstallCmd", func() {
		It("builds a cobra command", func() {
			cmd := newInstallCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(BeNil())
			Expect(cmd.Short).NotTo(BeNil())

			flag := cmd.Flags().Lookup("version")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal(installer.DefaultVersion))
			Expect(flag.Usage).NotTo(BeNil())
		})
	})
})
