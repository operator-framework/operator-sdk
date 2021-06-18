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

package bundle

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newFakeClient() returns a fake controller runtime client
func newFakeClient() client.Client {
	return fakeclient.NewClientBuilder().Build()
}

var _ = Describe("BundleInstall", func() {

	Describe("Bundle Test", func() {
		var (
			install Install
			cfg     *operator.Configuration
			sch     *runtime.Scheme
			// TODO bundleImage = "quay.io/example/example-operator-bundle:1.0.1"
			bundleImage = "quay.io/rashmigottipati/api-operator:1.0.1"
		)
		BeforeEach(func() {
			cfg = &operator.Configuration{}
			sch = runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			Expect(v1alpha1.AddToScheme(sch)).To(Succeed())
			cfg.Client = newFakeClient()
			install = NewInstall(cfg)
		})

		It("Should return an error with invalid bundle add mode", func() {
			install.BundleAddMode = "invalid"
			err := install.setup(context.TODO())
			Expect(err).ToNot(BeNil())
		})

		It("No error with valid bundle add mode", func() {
			install.BundleAddMode = index.SemverBundleAddMode
			install.BundleImage = bundleImage
			err := install.setup(context.TODO())
			Expect(err).To(BeNil())
			install.BundleAddMode = index.ReplacesBundleAddMode
			install.BundleImage = bundleImage
			err = install.setup(context.TODO())
			Expect(err).To(BeNil())
		})

		It("Fail with single namespace install mode", func() {

		})

		It("CheckCompatibility single namespace install mode", func() {
			mode := operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeSingleNamespace,
				TargetNamespaces: []string{"SingleNameSpace"},
			}
			install.InstallMode = mode
			install.BundleImage = bundleImage
			install.cfg.Namespace = "SingleNameSpace"
			err := install.setup(context.TODO())
			Expect(err).ToNot(BeNil())

			mode = operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeSingleNamespace,
				TargetNamespaces: []string{"SingleNameSpace"},
			}
			install.InstallMode = mode
			install.BundleImage = bundleImage
			install.cfg.Namespace = "OwnNamespace"
			err = install.setup(context.TODO())
			Expect(err).To(BeNil())
		})

		It("CheckCompatibility own namespace install mode", func() {
			mode := operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeOwnNamespace,
				TargetNamespaces: []string{"targetNs"},
			}
			install.InstallMode = mode
			install.BundleImage = bundleImage
			install.cfg.Namespace = "targetNs"
			err := install.setup(context.TODO())
			Expect(err).To(BeNil())
		})

		It("Should return an error with invalid bundle image name", func() {
			install.BundleImage = "dummy"
			err := install.setup(context.TODO())
			Expect(err).ToNot(BeNil())
		})

		It("Should not return an error with valid bundle image name", func() {
			install.BundleImage = bundleImage
			err := install.setup(context.TODO())
			Expect(err).To(BeNil())
		})
	})
})
