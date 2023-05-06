// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package registry

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("Configmap", func() {

	Describe("NewConfigMapCatalogCreator", func() {
		It("should return a configmapcreator instance", func() {
			cfg := operator.Configuration{
				Namespace: "testns",
			}

			ctlog := NewConfigMapCatalogCreator(&cfg)
			Expect(ctlog.cfg.Namespace).Should(Equal(cfg.Namespace))
		})
	})

	Describe("CreateCatalog", func() {
		It("should return an error if creation fails", func() {
			ctlog := &ConfigMapCatalogCreator{
				cfg: &operator.Configuration{
					Namespace: "testns",
					Client: fake.NewClientBuilder().WithObjects(
						newCatalogSource("pkgName", "testns", withSDKPublisher("pkgName")),
					).Build(),
				},
				Package: &apimanifests.PackageManifest{
					PackageName: "pkgName",
				},
			}

			x, err := ctlog.CreateCatalog(context.TODO(), "pkgName")
			Expect(err.Error()).Should(ContainSubstring("error creating catalog source"))
			Expect(x).Should(BeNil())
		})
	})

	Describe("updateCatalogSource", func() {
		It("should update the catalog source", func() {
			cs := newCatalogSource("pkgName", "testns", withSDKPublisher("pkgName"))
			ctlog := &ConfigMapCatalogCreator{
				cfg: &operator.Configuration{
					Namespace: "testns",
					Client:    fake.NewClientBuilder().WithObjects(cs).Build(),
				},
				Package: &apimanifests.PackageManifest{
					PackageName: "pkgName",
				},
			}
			expected := cs.DeepCopy()
			Expect(ctlog.updateCatalogSource(context.TODO(), cs)).Should(Succeed())
			Expect(expected.Spec.Address).ShouldNot(Equal(cs.Spec.Address))
			Expect(expected.Spec.SourceType).ShouldNot(Equal(cs.Spec.SourceType))
		})
	})
})
