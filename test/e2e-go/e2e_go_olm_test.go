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

package e2e_go_test

import (
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

var _ = Describe("Integrating Go Projects with OLM", func() {

	// OLM does not work with cert-manager. In this way, we need to generate
	// the pkg manifest after we comment cert-manager options into the kustomization file.
	// More info: https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/#certificate-authority-requirements
	BeforeEach(func() {
		By("commenting cert-manager")
		err := testutils.ReplaceInFile(
			filepath.Join(tc.Dir, "config", "default", "kustomization.yaml"),
			"- ../certmanager", "#- ../certmanager")
		Expect(err).NotTo(HaveOccurred())

		By("commenting cert-manager")
		err = testutils.ReplaceInFile(
			filepath.Join(tc.Dir, "config", "default", "kustomization.yaml"),
			uncommentedCertManager, commentedCertMaagerKustomizeFields)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("commenting cert-manager")
		err := testutils.ReplaceInFile(
			filepath.Join(tc.Dir, "config", "default", "kustomization.yaml"),
			"#- ../certmanager", "- ../certmanager")
		Expect(err).NotTo(HaveOccurred())

		By("commenting cert-manager")
		err = testutils.ReplaceInFile(
			filepath.Join(tc.Dir, "config", "default", "kustomization.yaml"),
			commentedCertMaagerKustomizeFields, uncommentedCertManager)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with operator-sdk", func() {
		const operatorVersion = "0.0.1"

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("adding the 'packagemanifests' rule to the Makefile")
			err := tc.AddPackagemanifestsTarget()
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator package manifests")
			err = tc.Make("packagemanifests", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("running the package manifests-formatted operator")
			Expect(err).NotTo(HaveOccurred())
			runPkgManCmd := exec.Command(tc.BinaryName, "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
				"--version", operatorVersion,
				"--timeout", "4m")
			_, err = tc.Run(runPkgManCmd)
			Expect(err).NotTo(HaveOccurred())

			By("destroying the deployed package manifests-formatted operator")
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", tc.ProjectName,
				"--timeout", "4m")
			_, err = tc.Run(cleanupPkgManCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

const uncommentedCertManager = `# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
- webhookcainjection_patch.yaml


# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1alpha2
    name: serving-cert # this name should match the one in certificate.yaml
  fieldref:
    fieldpath: metadata.namespace
- name: CERTIFICATE_NAME
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1alpha2
    name: serving-cert # this name should match the one in certificate.yaml
- name: SERVICE_NAMESPACE # namespace of the service
  objref:
    kind: Service
    version: v1
    name: webhook-service
  fieldref:
    fieldpath: metadata.namespace
- name: SERVICE_NAME
  objref:
    kind: Service
    version: v1
    name: webhook-service`

const commentedCertMaagerKustomizeFields = `# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
# - webhookcainjection_patch.yaml


# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service`
