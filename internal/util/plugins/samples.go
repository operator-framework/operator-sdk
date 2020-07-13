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

// TODO: rewrite this when plugins phase 2 is implemented.
package plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/config"

	"github.com/operator-framework/operator-sdk/internal/scaffold/kustomize"
)

// sampleKustomizationFragment is a template for samples/kustomization.yaml.
const sampleKustomizationFragment = `## This file is auto-generated, do not modify ##
resources:
`

// WriteSamplesKustomization perform the SDK plugin-specific scaffolds.
func WriteSamplesKustomization(cfg *config.Config) error {

	// Write CR paths to the samples' kustomization file. This file has a
	// "do not modify" comment so it is safe to overwrite.
	samplesKustomization := sampleKustomizationFragment
	for _, gvk := range cfg.Resources {
		samplesKustomization += fmt.Sprintf("- %s\n", makeCRFileName(gvk))
	}
	kpath := filepath.Join("config", "samples")
	if err := kustomize.Write(kpath, samplesKustomization); err != nil {
		return err
	}

	return nil
}

// todo(camilamacedo86): Now that we have the Kubebuilder scaffolding machinery included in our repo, we could make
// this an actual template that supports both file.Template and file.Inserter for init and create api, respectively.
// More info: https://github.com/operator-framework/operator-sdk/issues/3370
// makeCRFileName returns a Custom Resource example file name in the same format
// as kubebuilder's CreateAPI plugin for a gvk.
func makeCRFileName(gvk config.GVK) string {
	return fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
}
