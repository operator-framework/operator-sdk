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

package registry

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IndexImage", func() {
	Context("SecurityContext", func() {
		Describe("String", func() {
			It("should return a string value of the enum", func() {
				sc := SecurityContext{}
				Expect(sc.String()).To(Equal(""))
				sc = SecurityContext{ContextType: Legacy}
				Expect(sc.String()).To(Equal("legacy"))
			})
		})
		Describe("Set", func() {
			var sc SecurityContext
			BeforeEach(func() {
				sc = SecurityContext{}
				Expect(sc.String()).To(Equal(""))
			})
			It("should not error with valid values", func() {
				err := sc.Set("legacy")
				Expect(sc.String()).To(Equal("legacy"))
				Expect(err).ToNot(HaveOccurred())
				err = sc.Set("restricted")
				Expect(sc.String()).To(Equal("restricted"))
				Expect(err).ToNot(HaveOccurred())
			})
			It("should return an error with unsupported values", func() {
				err := sc.Set("fake")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("must be one of \"legacy\", or \"restricted\""))

				err = sc.Set("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("must be one of \"legacy\", or \"restricted\""))
			})
		})
		Describe("IsEmpty", func() {
			var sc SecurityContext
			BeforeEach(func() {
				sc = SecurityContext{}
				Expect(sc.String()).To(Equal(""))
			})
			It("should return true default instantiation", func() {
				Expect(sc.IsEmpty()).To(BeTrue())
			})
			It("should return false if set a value", func() {
				_ = sc.Set("legacy")
				Expect(sc.IsEmpty()).To(BeFalse())

				_ = sc.Set("restricted")
				Expect(sc.IsEmpty()).To(BeFalse())
			})
		})
		Describe("Type", func() {
			It("should return SecurityContext", func() {
				sc := SecurityContext{}
				Expect(sc.Type()).To(Equal("SecurityContext"))
			})
		})
	})
})
