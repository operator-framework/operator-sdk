// Copyright 2021 The Operator-SDK Authors
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

package util

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	log "github.com/sirupsen/logrus"
)

// RemoveKustomizeCRDManifests removes items in config/crd relating to CRD conversion webhooks.
func RemoveKustomizeCRDManifests() error {

	pathsToRemove := []string{
		filepath.Join("config", "crd", "kustomizeconfig.yaml"),
	}
	configPatchesDir := filepath.Join("config", "crd", "patches")
	webhookPatchMatches, err := filepath.Glob(filepath.Join(configPatchesDir, "webhook_in_*.yaml"))
	if err != nil {
		return err
	}
	pathsToRemove = append(pathsToRemove, webhookPatchMatches...)
	cainjectionPatchMatches, err := filepath.Glob(filepath.Join(configPatchesDir, "cainjection_in_*.yaml"))
	if err != nil {
		return err
	}
	pathsToRemove = append(pathsToRemove, cainjectionPatchMatches...)
	for _, p := range pathsToRemove {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	children, err := os.ReadDir(configPatchesDir)
	if err == nil && len(children) == 0 {
		if err := os.RemoveAll(configPatchesDir); err != nil {
			return err
		}
	}
	return nil
}

// UpdateKustomizationsInit updates certain parts of or removes entire kustomization.yaml files
// that are either not used by certain Init plugins or are created by preceding Init plugins.
func UpdateKustomizationsInit() error {

	defaultKFile := filepath.Join("config", "default", "kustomization.yaml")
	if err := kbutil.ReplaceInFile(defaultKFile,
		`
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
#- ../webhook
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'. 'WEBHOOK' components are required.
#- ../certmanager`, ""); err != nil {
		return fmt.Errorf("remove %s resources: %v", defaultKFile, err)
	}

	if err := kbutil.ReplaceInFile(defaultKFile,
		`
# [WEBHOOK] To enable webhook, uncomment all the sections with [WEBHOOK] prefix including the one in
# crd/kustomization.yaml
#- path: manager_webhook_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
#- path: webhookcainjection_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
# Uncomment the following replacements to add the cert-manager CA injection annotations
#replacements:
#  - source: # Add cert-manager annotation to ValidatingWebhookConfiguration, MutatingWebhookConfiguration and CRDs
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.namespace # namespace of the certificate CR
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#  - source:
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.name
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#  - source: # Add cert-manager annotation to the webhook Service
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.name # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 0
#          create: true
#  - source:
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.namespace # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 1
#          create: true
`, ""); err != nil {
		return fmt.Errorf("remove %s patch and vars blocks: %v", defaultKFile, err)
	}

	return nil
}

// UpdateKustomizationsCreateAPI updates certain parts of or removes entire kustomization.yaml files
// that are either not used by certain CreateAPI plugins or are created by preceding CreateAPI plugins.
func UpdateKustomizationsCreateAPI() error {

	crdKFile := filepath.Join("config", "crd", "kustomization.yaml")
	if crdKBytes, err := os.ReadFile(crdKFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Debugf("Error reading kustomization for substitution: %v", err)
	} else if err == nil {
		if bytes.Contains(crdKBytes, []byte("[WEBHOOK]")) || bytes.Contains(crdKBytes, []byte("[CERTMANAGER]")) {
			if err := os.RemoveAll(crdKFile); err != nil {
				log.Debugf("Error removing file prior to scaffold: %v", err)
			}
		}
	}

	return nil
}
