// Copyright 2018 The Operator-SDK Authors
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

package genutil

import (
	"fmt"
	"path/filepath"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
)

// CRDGen generates CRDs for all APIs in pkg/apis.
func CRDGen(crdVersion string) error {
	projutil.MustInProjectRoot()

	log.Info("Running CRD generator.")

	crd := gencrd.Generator{
		IsOperatorGo: true,
		CRDVersion:   crdVersion,
	}
	if err := crd.Generate(); err != nil {
		return fmt.Errorf("error generating CRDs from APIs in %s: %w", scaffold.ApisDir, err)
	}

	log.Info("CRD generation complete.")
	return nil
}

// GenerateCRDNonGo generates CRDs for Non-Go APIs(Eg., Ansible,Helm)
func GenerateCRDNonGo(projectName string, resource scaffold.Resource, crdVersion string) error {
	crdsDir := filepath.Join(projectName, scaffold.CRDsDir)
	crd := gencrd.Generator{
		CRDsDir:      crdsDir,
		OutputDir:    crdsDir,
		CRDVersion:   crdVersion,
		Resource:     resource,
		IsOperatorGo: false,
	}
	if err := crd.Generate(); err != nil {
		return fmt.Errorf("error generating CRD for %s: %w", resource, err)
	}
	log.Info("Generated CustomResourceDefinition manifests.")
	return nil
}
