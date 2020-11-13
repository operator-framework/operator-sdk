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

package gosamples

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	kbtestutils "sigs.k8s.io/kubebuilder/v2/test/e2e/utils"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// RunV2 the steps to create the Memcached with Webhooks Go Sample
func (mh *MemcachedGoWithWebhooks) RunV3() {
	log.Infof("creating the project")
	err := mh.ctx.Init(
		"--plugins", "go/v3-alpha",
		"--repo", "github.com/example/memcached-operator",
		"--domain",
		mh.ctx.Domain)
	pkg.CheckError("creating the project", err)

	err = mh.ctx.CreateAPI(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--controller", "true",
		"--resource", "true")
	pkg.CheckError("scaffolding apis", err)

	log.Infof("implementing the API")
	mh.implementingAPI()

	log.Infof("implementing the Controller")
	mh.implementingControllerV3()

	log.Infof("scaffolding webhook")
	err = mh.ctx.CreateWebhook(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--defaulting",
		"--defaulting")
	pkg.CheckError("scaffolding webhook", err)

	mh.implementingWebhooks()
	mh.uncommentKustomizationFile()

	mh.ctx.CreateBundle()

	// Clean up built binaries, if any.
	pkg.CheckError("cleaning up", os.RemoveAll(filepath.Join(mh.ctx.Dir, "bin")))
}

// GenerateMemcachedGoWithWebhooksSampleV2 will call all actions to create the directory and generate the sample
// Note that it should NOT be called in the e2e tests.
func GenerateMemcachedGoWithWebhooksSampleV3(samplesPath string) {
	log.Infof("starting to generate Go memcached sample with webhooks")
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(samplesPath, "go", "v3", "memcached-operator"), "GO111MODULE=on")
	pkg.CheckError("generating Go memcached with webhooks context", err)

	memcached := NewMemcachedGoWithWebhooks(&ctx)
	memcached.Prepare()
	memcached.RunV3()
}

// implementingController will customize the Controller
func (mh *MemcachedGoWithWebhooks) implementingControllerV3() {
	controllerPath := filepath.Join(mh.ctx.Dir, "controllers", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	// Add imports
	log.Infof("adding imports")
	err := kbtestutils.InsertCode(controllerPath,
		"import (",
		importsFragment)
	pkg.CheckError("adding imports", err)

	// Add RBAC permissions on top of reconcile
	log.Infof("adding RBAC permissions on top of reconcile")
	err = kbtestutils.InsertCode(controllerPath,
		"verbs=get;update;patch",
		rbacFragmentV3)
	pkg.CheckError("adding rbac", err)

	err = testutils.ReplaceInFile(controllerPath,
		fmt.Sprintf("_ = r.Log.WithValues(\"%s\", req.NamespacedName)", strings.ToLower(mh.ctx.Kind)),
		fmt.Sprintf("log := r.Log.WithValues(\"%s\", req.NamespacedName)", strings.ToLower(mh.ctx.Kind)))
	pkg.CheckError("replacing reconcile content", err)

	// Add reconcile implementation
	err = testutils.ReplaceInFile(controllerPath,
		"// your logic here", reconcileFragment)
	pkg.CheckError("replacing reconcile", err)

	// Add helpers funcs to the controller
	err = kbtestutils.InsertCode(controllerPath,
		"return ctrl.Result{}, nil\n}", controllerFuncsFragment)
	pkg.CheckError("adding helpers methods in the controller", err)

	// Add watch for the Kind
	err = testutils.ReplaceInFile(controllerPath,
		fmt.Sprintf(watchOriginalFragment, mh.ctx.Group, mh.ctx.Version, mh.ctx.Kind),
		fmt.Sprintf(watchCustomizedFragment, mh.ctx.Group, mh.ctx.Version, mh.ctx.Kind))
	pkg.CheckError("replacing reconcile", err)
}

const rbacFragmentV3 = `
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;`
