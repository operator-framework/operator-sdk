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

package operator

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

var _ = Describe("InstallMode", func() {

	Describe("GetSupportedInstallModes", func() {
		It("should return empty set if empty installmodes", func() {
			supported := GetSupportedInstallModes([]v1alpha1.InstallMode{})
			Expect(supported.Len()).To(Equal(0))
		})
		It("should return empty set if no installmodes are supported", func() {
			installModes := []v1alpha1.InstallMode{
				{
					Type:      v1alpha1.InstallModeTypeSingleNamespace,
					Supported: false,
				},
				{
					Type:      v1alpha1.InstallModeTypeOwnNamespace,
					Supported: false,
				},
			}
			supported := GetSupportedInstallModes(installModes)
			Expect(supported.Len()).To(Equal(0))
		})
		It("should return set with supported installmodes", func() {
			installModes := []v1alpha1.InstallMode{
				{
					Type:      v1alpha1.InstallModeTypeSingleNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeOwnNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeAllNamespaces,
					Supported: false,
				},
			}
			supported := GetSupportedInstallModes(installModes)
			Expect(supported.Len()).To(Equal(2))
			Expect(supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace))).Should(BeTrue())
			Expect(supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace))).Should(BeTrue())
			Expect(supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces))).Should(BeFalse())
		})
	})
})
