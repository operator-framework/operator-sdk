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

package bases

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

func TestMetadata(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metadata Suite")
}

var _ = Describe("Metadata", func() {

	meta := uiMetadata{
		DisplayName: "Memcached Application",
		Keywords:    []string{"memcached", "app"},
		Description: "Main enterprise application providing business critical features with " +
			"high availability and no manual intervention.",
		ProviderName: "Example",
		ProviderURL:  "www.example.com",
		Maintainers:  []string{"Some Corp:corp@example.com"},
	}

	It("populates an empty CSV", func() {
		csv := v1alpha1.ClusterServiceVersion{}

		meta.apply(&csv)

		Expect(csv.Spec.DisplayName).To(Equal(meta.DisplayName))
		Expect(csv.Spec.Keywords).To(Equal(meta.Keywords))
		Expect(csv.Spec.Description).To(Equal(meta.Description))
		Expect(csv.Spec.Maintainers).To(Equal([]v1alpha1.Maintainer{{Name: "Some Corp", Email: "corp@example.com"}}))
		Expect(csv.Spec.Provider).To(Equal(v1alpha1.AppLink{Name: meta.ProviderName, URL: meta.ProviderURL}))
	})

	It("populates a CSV with existing values", func() {
		b := ClusterServiceVersion{OperatorName: "test-operator"}
		b.setDefaults()
		csv := b.newBase()

		meta.apply(csv)

		Expect(csv.Spec.DisplayName).To(Equal(meta.DisplayName))
		Expect(csv.Spec.Keywords).To(Equal(meta.Keywords))
		Expect(csv.Spec.Description).To(Equal(meta.Description))
		Expect(csv.Spec.Maintainers).To(Equal([]v1alpha1.Maintainer{{Name: "Some Corp", Email: "corp@example.com"}}))
		Expect(csv.Spec.Provider).To(Equal(v1alpha1.AppLink{Name: meta.ProviderName, URL: meta.ProviderURL}))
	})
})
