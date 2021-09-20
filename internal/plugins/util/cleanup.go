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
	"io/ioutil"
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
	children, err := ioutil.ReadDir(configPatchesDir)
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
#- manager_webhook_patch.yaml

# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER'.
# Uncomment 'CERTMANAGER' sections in crd/kustomization.yaml to enable the CA injection in the admission webhooks.
# 'CERTMANAGER' needs to be enabled to use ca injection
#- webhookcainjection_patch.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
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
#    name: webhook-service
`, ""); err != nil {
		return fmt.Errorf("remove %s patch and vars blocks: %v", defaultKFile, err)
	}

	return nil
}

// UpdateKustomizationsCreateAPI updates certain parts of or removes entire kustomization.yaml files
// that are either not used by certain CreateAPI plugins or are created by preceding CreateAPI plugins.
func UpdateKustomizationsCreateAPI() error {

	crdKFile := filepath.Join("config", "crd", "kustomization.yaml")
	if crdKBytes, err := ioutil.ReadFile(crdKFile); err != nil && !errors.Is(err, os.ErrNotExist) {
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
