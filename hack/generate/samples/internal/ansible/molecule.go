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
	"os/exec"
	"path/filepath"
	"strings"

	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	log "github.com/sirupsen/logrus"
)

// MoleculeAnsible defines the context for the sample
type MoleculeAnsible struct {
	ctx *pkg.SampleContext
}

// NewMoleculeAnsible return a MoleculeAnsible
func NewMoleculeAnsible(ctx *pkg.SampleContext) MoleculeAnsible {
	return MoleculeAnsible{ctx}
}

// Prepare the Context for the Memcached Ansible Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (ma *MoleculeAnsible) Prepare() {
	log.Infof("destroying directory for memcached Ansible samples")
	ma.ctx.Destroy()

	log.Infof("creating directory")
	err := ma.ctx.Prepare()
	pkg.CheckError("creating directory for Ansible Sample", err)

	log.Infof("setting domain and GVK")
	ma.ctx.Domain = "example.com"
	ma.ctx.Version = "v1alpha1"
	ma.ctx.Group = "cache"
	ma.ctx.Kind = "Memcached"
}

// Run the steps to create the Memcached Ansible Sample
func (ma *MoleculeAnsible) Run() {
	memcached := NewMemcachedAnsible(ma.ctx)
	memcached.Run()

	moleculeTaskPath := filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks",
		fmt.Sprintf("%s_test.yml", strings.ToLower(ma.ctx.Kind)))

	log.Infof("insert molecule task to ensure that ConfigMap will be deleted")
	err := kbtestutils.InsertCode(moleculeTaskPath, targetMoleculeCheckDeployment, molecuTaskToCheckConfigMap)
	pkg.CheckError("replacing memcached task to add config map check", err)

	log.Infof("insert molecule task to ensure to check secret")
	err = kbtestutils.InsertCode(moleculeTaskPath, memcachedCustomStatusMoleculeTarget, testSecretMoleculeCheck)
	pkg.CheckError("replacing memcached task to add secret check", err)

	log.Infof("insert molecule task to ensure to foo ")
	err = kbtestutils.InsertCode(moleculeTaskPath, testSecretMoleculeCheck, testFooMoleculeCheck)
	pkg.CheckError("replacing memcached task to add foo check", err)

	log.Infof("replacing project Dockerfile to use ansible base image with the dev tag")
	err = testutils.ReplaceRegexInFile(filepath.Join(ma.ctx.Dir, "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	pkg.CheckError("replacing Dockerfile", err)

	log.Infof("adding RBAC permissions")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "config", "rbac", "role.yaml"),
		"# +kubebuilder:scaffold:rules", rolesForBaseOperator)
	pkg.CheckError("replacing in role.yml", err)

	log.Infof("adding Memcached mock task to the role with black list")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind), "tasks", "main.yml"),
		roleFragment, memcachedWithBlackListTask)
	pkg.CheckError("replacing in tasks/main.yml", err)

	log.Infof("creating an API definition Foo")
	err = ma.ctx.CreateAPI(
		"--group", ma.ctx.Group,
		"--version", ma.ctx.Version,
		"--kind", "Foo",
		"--generate-role")
	pkg.CheckError("creating api", err)

	log.Infof("creating an API definition to add a task to delete the config map")
	err = ma.ctx.CreateAPI(
		"--group", ma.ctx.Group,
		"--version", ma.ctx.Version,
		"--kind", "Memfin",
		"--generate-role")
	pkg.CheckError("creating api", err)

	log.Infof("adding task to delete config map")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", "memfin", "tasks", "main.yml"),
		"# tasks file for Memfin", taskToDeleteConfigMap)
	pkg.CheckError("replacing in tasks/main.yml", err)

	log.Infof("adding to watches finalizer and blacklist")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "watches.yaml"),
		"playbook: playbooks/memcached.yml", memcachedWatchCustomizations)
	pkg.CheckError("replacing in watches", err)

	log.Infof("enabling multigroup support")
	err = ma.ctx.AllowProjectBeMultiGroup()
	pkg.CheckError("updating PROJECT file", err)

	log.Infof("creating core Secret API")
	err = ma.ctx.CreateAPI(
		// the tool do not allow we crate an API with a group nil for v2+
		// which is required here to mock the tests.
		// however, it is done already for v3+. More info: https://github.com/kubernetes-sigs/kubebuilder/issues/1404
		// and the tests should be changed when the tool allows we create API's for core types.
		// todo: replace the ignore value when the tool provide a solution for it.
		"--group", "ignore",
		"--version", "v1",
		"--kind", "Secret",
		"--generate-role")
	pkg.CheckError("creating api", err)

	log.Infof("removing ignore group for the secret from watches as an workaround to work with core types")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "watches.yaml"),
		"ignore.example.com", "\"\"")
	pkg.CheckError("replacing the watches file", err)

	log.Infof("removing molecule test for the Secret since it is a core type")
	cmd := exec.Command("rm", "-rf", filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks", "secret_test.yml"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("removing secret test file", err)

	log.Infof("adding Secret task to the role")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", "secret", "tasks", "main.yml"),
		originalTaskSecret, taskForSecret)
	pkg.CheckError("replacing in secret/tasks/main.yml file", err)

	log.Infof("adding ManageStatus == false for role secret")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "watches.yaml"),
		"role: secret", manageStatusFalseForRoleSecret)
	pkg.CheckError("replacing in watches.yaml", err)

	log.Infof("removing FIXME asserts from memfin_test.yml")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks", "memfin_test.yml"),
		fixmeAssert, "")
	pkg.CheckError("replacing memfin_test.yml", err)

	log.Infof("removing FIXME asserts from foo_test.yml")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks", "foo_test.yml"),
		fixmeAssert, "")
	pkg.CheckError("replacing foo_test.yml", err)
}

// GenerateMoleculeAnsibleSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMoleculeAnsibleSample(path string) {
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(path, "memcached-molecule-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Ansible Moleule memcached context", err)

	molecule := NewMoleculeAnsible(&ctx)
	molecule.Prepare()
	molecule.Run()
}
