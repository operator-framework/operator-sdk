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
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/util"
	"github.com/operator-framework/operator-sdk/testutils/e2e"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

func ImplementMemcachedMolecule(sample sample.Sample, image string) {

	for _, gvk := range sample.GVKs() {
		moleculeTaskPath := filepath.Join(sample.Dir(), "molecule", "default", "tasks",
			fmt.Sprintf("%s_test.yml", strings.ToLower(gvk.Kind)))

		log.Infof("insert molecule task to ensure that ConfigMap will be deleted")
		err := kbutil.InsertCode(moleculeTaskPath, targetMoleculeCheckDeployment, molecuTaskToCheckConfigMap)
		pkg.CheckError("replacing memcached task to add config map check", err)

		log.Infof("insert molecule task to ensure to check secret")
		err = kbutil.InsertCode(moleculeTaskPath, memcachedCustomStatusMoleculeTarget, testSecretMoleculeCheck)
		pkg.CheckError("replacing memcached task to add secret check", err)

		log.Infof("insert molecule task to ensure to foo ")
		err = kbutil.InsertCode(moleculeTaskPath, testSecretMoleculeCheck, testFooMoleculeCheck)
		pkg.CheckError("replacing memcached task to add foo check", err)

		log.Infof("insert molecule task to check custom metrics")
		err = kbutil.InsertCode(moleculeTaskPath, testFooMoleculeCheck, customMetricsTest)

		pkg.CheckError("replacing memcached task to add foo check", err)

		log.Infof("adding Memcached mock task to the role with black list")
		err = kbutil.InsertCode(filepath.Join(sample.Dir(), "roles", strings.ToLower(gvk.Kind), "tasks", "main.yml"),
			roleFragment, memcachedWithBlackListTask)
		pkg.CheckError("replacing in tasks/main.yml", err)

		log.Infof("updating spec of kind: ", gvk.Kind)
		err = kbutil.ReplaceInFile(
			filepath.Join(sample.Dir(), "config", "samples", fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))),
			"# TODO(user): Add fields here",
			"foo: bar")
		pkg.CheckError("updating spec of "+fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind)), err)

		log.Infof("removing FIXME asserts from %s_test.yml", strings.ToLower(gvk.Kind))
		err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "molecule", "default", "tasks", fmt.Sprintf("%s_test.yml", strings.ToLower(gvk.Kind))),
			fixmeAssert, "")
		pkg.CheckError(fmt.Sprintf("replacing %s_test.yml", strings.ToLower(gvk.Kind)), err)
	}

	log.Infof("replacing project Dockerfile to use ansible base image with the dev tag")
	err := util.ReplaceRegexInFile(filepath.Join(sample.Dir(), "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	pkg.CheckError("replacing Dockerfile", err)

	log.Infof("adding RBAC permissions")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "config", "rbac", "role.yaml"),
		"#+kubebuilder:scaffold:rules", rolesForBaseOperator)
	pkg.CheckError("replacing in role.yml", err)

	log.Infof("adding task to delete config map")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "roles", "memfin", "tasks", "main.yml"),
		"# tasks file for Memfin", taskToDeleteConfigMap)
	pkg.CheckError("replacing in tasks/main.yml", err)

	log.Infof("adding to watches finalizer and blacklist")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "watches.yaml"),
		"playbook: playbooks/memcached.yml", memcachedWatchCustomizations)
	pkg.CheckError("replacing in watches", err)

	log.Infof("enabling multigroup support")
	err = e2e.AllowProjectBeMultiGroup(sample)
	pkg.CheckError("updating PROJECT file", err)

	log.Infof("removing ignore group for the secret from watches as an workaround to work with core types")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "watches.yaml"),
		"ignore.example.com", "\"\"")
	pkg.CheckError("replacing the watches file", err)

	log.Infof("removing molecule test for the Secret since it is a core type")
	cmd := exec.Command("rm", "-rf", filepath.Join(sample.Dir(), "molecule", "default", "tasks", "secret_test.yml"))
	_, err = sample.CommandContext().Run(cmd)
	pkg.CheckError("removing secret test file", err)

	log.Infof("adding Secret task to the role")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "roles", "secret", "tasks", "main.yml"),
		originalTaskSecret, taskForSecret)
	pkg.CheckError("replacing in secret/tasks/main.yml file", err)

	log.Infof("adding ManageStatus == false for role secret")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "watches.yaml"),
		"role: secret", manageStatusFalseForRoleSecret)
	pkg.CheckError("replacing in watches.yaml", err)

	// prevent high load of controller caused by watching all the secrets in the cluster
	watchNamespacePatchFileName := "watch_namespace_patch.yaml"
	log.Info("adding WATCH_NAMESPACE env patch to watch own namespace")
	err = ioutil.WriteFile(filepath.Join(sample.Dir(), "config", "testing", watchNamespacePatchFileName), []byte(watchNamespacePatch), 0644)
	pkg.CheckError("adding watch_namespace_patch.yaml", err)

	log.Info("adding WATCH_NAMESPACE env patch to patch list to be applied")
	err = kbutil.InsertCode(filepath.Join(sample.Dir(), "config", "testing", "kustomization.yaml"), "patchesStrategicMerge:",
		fmt.Sprintf("\n- %s", watchNamespacePatchFileName))
	pkg.CheckError("inserting in kustomization.yaml", err)
}
