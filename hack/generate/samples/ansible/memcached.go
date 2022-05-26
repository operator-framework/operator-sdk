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

package ansible

import (
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

func ImplementMemcached(sample sample.Sample, image string) {
	log.Infof("customizing the sample")
	err := kbutil.UncommentCode(
		filepath.Join(sample.Dir(), "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	for _, gvk := range sample.GVKs() {
		addingAnsibleTask(sample.Dir(), gvk)
		addingMoleculeMockData(sample.Dir(), sample.Name(), gvk)
	}

	log.Infof("creating the bundle")
	err = olm.GenerateBundle(sample, image)
	pkg.CheckError("creating the bundle", err)

	log.Infof("striping bundle annotations")
	err = olm.StripBundleAnnotations(sample)
	pkg.CheckError("striping bundle annotations", err)
}

// addingMoleculeMockData will customize the molecule data
func addingMoleculeMockData(dir string, projectName string, gvk schema.GroupVersionKind) {
	log.Infof("adding molecule test for Ansible task")
	moleculeTaskPath := filepath.Join(dir, "molecule", "default", "tasks",
		fmt.Sprintf("%s_test.yml", strings.ToLower(gvk.Kind)))

	err := kbutil.ReplaceInFile(moleculeTaskPath,
		originaMemcachedMoleculeTask, fmt.Sprintf(moleculeTaskFragment, projectName, projectName))
	pkg.CheckError("replacing molecule default tasks", err)
}

// addingAnsibleTask will add the Ansible Task and update the sample
func addingAnsibleTask(dir string, gvk schema.GroupVersionKind) {
	log.Infof("adding Ansible task and variable")
	err := kbutil.InsertCode(filepath.Join(dir, "roles", strings.ToLower(gvk.Kind),
		"tasks", "main.yml"),
		fmt.Sprintf("# tasks file for %s", gvk.Kind),
		roleFragment)
	pkg.CheckError("adding task", err)

	err = kbutil.ReplaceInFile(filepath.Join(dir, "roles", strings.ToLower(gvk.Kind),
		"defaults", "main.yml"),
		fmt.Sprintf("# defaults file for %s", gvk.Kind),
		defaultsFragment)
	pkg.CheckError("adding defaulting", err)

	err = kbutil.ReplaceInFile(filepath.Join(dir, "config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))),
		"# TODO(user): Add fields here", "size: 1")
	pkg.CheckError("updating sample CR", err)
}
