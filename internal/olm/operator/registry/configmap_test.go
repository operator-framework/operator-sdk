package registry

import (
	"context"

	. "github.com/onsi/ginkgo"
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
					Client:    fake.NewFakeClient(newCatalogSource("pkgName", "testns", withSDKPublisher("pkgName"))),
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
					Client:    fake.NewFakeClient(cs),
				},
				Package: &apimanifests.PackageManifest{
					PackageName: "pkgName",
				},
			}
			expected := cs.DeepCopy()
			err := ctlog.updateCatalogSource(context.TODO(), cs)

			Expect(err).Should(BeNil())
			Expect(expected.Spec.Address).ShouldNot(Equal(cs.Spec.Address))
			Expect(expected.Spec.SourceType).ShouldNot(Equal(cs.Spec.SourceType))
		})
	})
})
