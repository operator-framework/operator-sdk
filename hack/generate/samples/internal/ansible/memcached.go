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

	log "github.com/sirupsen/logrus"
	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// MemcachedAnsible defines the context for the sample
type MemcachedAnsible struct {
	ctx *pkg.SampleContext
}

// NewMemcachedAnsible return a MemcachedAnsible
func NewMemcachedAnsible(ctx *pkg.SampleContext) MemcachedAnsible {
	return MemcachedAnsible{ctx}
}

// Prepare the Context for the Memcached Ansible Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (ma *MemcachedAnsible) Prepare() {
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
func (ma *MemcachedAnsible) Run() {
	log.Infof("creating the project")
	err := ma.ctx.Init(
		"--plugins", "ansible",
		"--group", ma.ctx.Group,
		"--version", ma.ctx.Version,
		"--kind", ma.ctx.Kind,
		"--domain", ma.ctx.Domain,
		"--generate-role",
		"--generate-playbook")
	pkg.CheckError("creating the project", err)

	log.Infof("customizing the sample")
	err = testutils.UncommentCode(
		filepath.Join(ma.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	log.Infof("adding Memcached mock task to the role with black list")
	err = kbtestutils.InsertCode(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"tasks", "main.yml"),
		fmt.Sprintf("# tasks file for %s", ma.ctx.Kind),
		memcachedWithBlackListTask)
	pkg.CheckError("adding task", err)

	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"defaults", "main.yml"),
		fmt.Sprintf("# defaults file for %s", ma.ctx.Kind),
		defaultsFragment)
	pkg.CheckError("adding defaulting", err)

	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", ma.ctx.Group, ma.ctx.Version, strings.ToLower(ma.ctx.Kind))),
		"foo: bar", "size: 1")
	pkg.CheckError("updating sample CR", err)

	ma.addingMoleculeMockData()

	log.Infof("adding RBAC permissions")
	err = testutils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "config", "rbac", "role.yaml"),
		"# +kubebuilder:scaffold:rules", rolesForBaseOperator)
	pkg.CheckError("replacing in role.yml", err)

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

	pkg.RunOlmIntegration(ma.ctx)
}

// addingMoleculeMockData will customize the molecule data
func (ma *MemcachedAnsible) addingMoleculeMockData() {
	log.Infof("adding molecule test for Ansible task")
	moleculeTaskPath := filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks",
		fmt.Sprintf("%s_test.yml", strings.ToLower(ma.ctx.Kind)))

	err := testutils.ReplaceInFile(moleculeTaskPath,
		originaMemcachedMoleculeTask, fmt.Sprintf(moleculeTaskFragment, ma.ctx.ProjectName, ma.ctx.ProjectName))
	pkg.CheckError("replacing molecule default tasks", err)

	log.Infof("insert molecule task to ensure that ConfigMap will be deleted")
	err = kbtestutils.InsertCode(moleculeTaskPath, targetMoleculeCheckDeployment, molecuTaskToCheckConfigMap)
	pkg.CheckError("replacing memcached task to add config map check", err)

	log.Infof("insert molecule task to ensure to check secret")
	err = kbtestutils.InsertCode(moleculeTaskPath, memcachedCustomStatusMoleculeTarget, testSecretMoleculeCheck)
	pkg.CheckError("replacing memcached task to add secret check", err)

	log.Infof("insert molecule task to ensure to foo ")
	err = kbtestutils.InsertCode(moleculeTaskPath, testSecretMoleculeCheck, testFooMoleculeCheck)
	pkg.CheckError("replacing memcached task to add foo check", err)
}

// GenerateMemcachedAnsibleSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMemcachedAnsibleSample(samplesPath string) {
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(samplesPath, "ansible", "memcached-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Ansible memcached context", err)

	memcached := NewMemcachedAnsible(&ctx)
	memcached.Prepare()
	memcached.Run()
}
