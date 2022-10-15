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

var _ = Describe("newCatalogSource", func() {
	Describe("newCatalogSource", func() {
		It("should create a CatalogSource with name and namespace set", func() {
			cs := newCatalogSource("fakeName", "fakeNS")
			Expect(cs.ObjectMeta.Name).To(Equal("fakeName"))
			Expect(cs.ObjectMeta.Namespace).To(Equal("fakeNS"))
		})
	})
	Describe("withSDKPublisher", func() {
		It("should set the display name and publisher of a CatalogSource", func() {
			cs := newCatalogSource("fakeName", "fakeNS", withSDKPublisher("fakeDisplay"))
			Expect(cs.Spec.DisplayName).To(Equal("fakeDisplay"))
			Expect(cs.Spec.Publisher).To(Equal("operator-sdk"))
		})
	})
	Describe("withInstallPlanApproval", func() {
		It("should set the display name and publisher of a CatalogSource", func() {
			cs := newCatalogSource("fakeName", "fakeNS", withSDKPublisher("fakeDisplay"))
			Expect(cs.Spec.DisplayName).To(Equal("fakeDisplay"))
			Expect(cs.Spec.Publisher).To(Equal("operator-sdk"))
		})
	})

})
