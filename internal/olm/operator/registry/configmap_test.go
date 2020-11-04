package registry

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("Configmap", func() {

	Describe("NewConfigMapCatalogCreator", func() {
		It("should return a configmapcreator with a configuration", func() {
			cfg := operator.Configuration{
				Namespace: "testns",
			}

			ctlog := NewConfigMapCatalogCreator(&cfg)
			Expect(ctlog.cfg.Namespace).Should(Equal(cfg.Namespace))
		})
	})

	Describe("CreateCatalog", func() {
		It("should return a configmapcreator with a configuration", func() {
			cfg := operator.Configuration{
				Namespace: "testns",
				Client:    nil,
			}
			cfg.Client = fake.NewFakeClient(newCatalogSource("pkgName", "testns", withSDKPublisher("pkgName")))
			ctlog := NewConfigMapCatalogCreator(&cfg)
			ctlog.Package = &apimanifests.PackageManifest{
				PackageName: "pkgName",
			}

			x, err := ctlog.CreateCatalog(context.TODO(), "pkgName")
			Expect(err.Error()).Should(ContainSubstring("already exists"))
			Expect(x).Should(BeNil())
		})
	})

	Describe("updateCatalogSource", func() {
		It("should update the catalog source", func() {
			cfg := operator.Configuration{
				Namespace: "testns",
				Client:    nil,
			}
			cfg.Client = fake.NewFakeClient(newCatalogSource("pkgName", "testns", withSDKPublisher("pkgName")))
			ctlog := NewConfigMapCatalogCreator(&cfg)
			ctlog.Package = &apimanifests.PackageManifest{
				PackageName: "pkgName",
			}
			x := v1alpha1.CatalogSource{
				ObjectMeta: v1.ObjectMeta{
					Name: "pkgName",
				},
			}
			y := x
			err := ctlog.updateCatalogSource(context.TODO(), &x)

			Expect(err).Should(BeNil())
			Expect(y).ShouldNot(Equal(x))
		})
	})
})
