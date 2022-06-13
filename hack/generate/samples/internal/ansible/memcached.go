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
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
)

// Memcached defines the context for the sample
type Memcached struct {
	ctx *pkg.SampleContext
}

// GenerateMemcachedSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMemcachedSample(binaryPath, samplesPath string) {
	ctx, err := pkg.NewSampleContext(binaryPath, filepath.Join(samplesPath, "memcached-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Ansible memcached context", err)

	memcached := Memcached{&ctx}
	memcached.Prepare()
	memcached.Run()
}

// Prepare the Context for the Memcached Ansible Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (ma *Memcached) Prepare() {
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
func (ma *Memcached) Run() {
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
	err = kbutil.UncommentCode(
		filepath.Join(ma.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	ma.ctx.UncommentRestrictivePodStandards()
	ma.addingAnsibleTask()
	ma.addingMoleculeMockData()

	log.Infof("creating the bundle")
	err = ma.ctx.GenerateBundle()
	pkg.CheckError("creating the bundle", err)

	log.Infof("striping bundle annotations")
	err = ma.ctx.StripBundleAnnotations()
	pkg.CheckError("striping bundle annotations", err)
}

// addingMoleculeMockData will customize the molecule data
func (ma *Memcached) addingMoleculeMockData() {
	log.Infof("adding molecule test for Ansible task")
	moleculeTaskPath := filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks",
		fmt.Sprintf("%s_test.yml", strings.ToLower(ma.ctx.Kind)))

	err := kbutil.ReplaceInFile(moleculeTaskPath,
		originaMemcachedMoleculeTask, fmt.Sprintf(moleculeTaskFragment, ma.ctx.ProjectName, ma.ctx.ProjectName))
	pkg.CheckError("replacing molecule default tasks", err)
}

// addingAnsibleTask will add the Ansible Task and update the sample
func (ma *Memcached) addingAnsibleTask() {
	log.Infof("adding Ansible task and variable")
	err := kbutil.InsertCode(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"tasks", "main.yml"),
		fmt.Sprintf("# tasks file for %s", ma.ctx.Kind),
		roleFragment)
	pkg.CheckError("adding task", err)

	err = kbutil.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"defaults", "main.yml"),
		fmt.Sprintf("# defaults file for %s", ma.ctx.Kind),
		defaultsFragment)
	pkg.CheckError("adding defaulting", err)

	err = kbutil.ReplaceInFile(filepath.Join(ma.ctx.Dir, "config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", ma.ctx.Group, ma.ctx.Version, strings.ToLower(ma.ctx.Kind))),
		"# TODO(user): Add fields here", "size: 1")
	pkg.CheckError("updating sample CR", err)
}
