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

package internal

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/operator"
)

func TestOperatorInstaller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test OperatorInstaller Suite")
}

var _ = Describe("Operator Installer", func() {

	Describe("creating subscription", func() {

		Context("with valid values", func() {
			var sub *v1alpha1.Subscription
			config := &operator.Configuration{
				Client:    fakeclient.NewFakeClient(),
				Namespace: "test-default-namespace",
			}

			oi := &OperatorInstaller{
				cfg:               config,
				CatalogSourceName: "test-cs",
				PackageName:       "test-package",
				StartingCSV:       "test-csv",
				Channel:           "test-default-channel",
			}

			BeforeEach(func() {
				sub = oi.createSubscription()
				Expect(sub).NotTo(BeNil())
			})

			It("should create subscription successfully", func() {
				Expect(sub.Name).Should(Equal(oi.CatalogSourceName + "-sub"))
				Expect(sub.Namespace).Should(Equal(oi.cfg.Namespace))
				Expect(sub.Spec.InstallPlanApproval).NotTo(BeNil())
				Expect(sub.Spec.InstallPlanApproval).Should(Equal(v1alpha1.ApprovalManual))
				Expect(sub.Spec.CatalogSource).Should(Equal(oi.CatalogSourceName))
				Expect(sub.Spec.CatalogSourceNamespace).Should(Equal(oi.cfg.Namespace))
				Expect(sub.Spec.Channel).Should(Equal(oi.Channel))
				Expect(sub.Spec.StartingCSV).Should(Equal(oi.StartingCSV))
				Expect(sub.Spec.Package).Should(Equal(oi.PackageName))
			})

		})
	})
})
