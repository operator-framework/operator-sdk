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

package packagemanifests

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Creating a generate packagemanifests command", func() {
	Describe("NewCmd", func() {
		It("builds and returns a cobra command", func() {
			cmd := NewCmd()
			Expect(cmd).NotTo(BeNil())

			flag := cmd.Flags().Lookup("version")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("v"))
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("from-version")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("input-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("output-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("kustomize-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal(filepath.Join("config", "manifests")))
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("deploy-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("crds-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("channel")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("default-channel")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("false"))
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("update-objects")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("true"))
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("quiet")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("q"))
			Expect(flag.DefValue).To(Equal("false"))
			Expect(flag.Usage).ToNot(Equal(""))

			flag = cmd.Flags().Lookup("stdout")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("false"))
			Expect(flag.Usage).ToNot(Equal(""))
		})
	})
})
