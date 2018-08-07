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

package test

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

const (
	ProjRootFlag    = "root"
	KubeConfigFlag  = "kubeconfig"
	CrdManPathFlag  = "crd"
	OpManPathFlag   = "op"
	RbacManPathFlag = "rbac"
)

func MainEntry(m *testing.M) {
	projRoot := flag.String("root", "", "path to project root")
	kubeconfigPath := flag.String("kubeconfig", "", "path to kubeconfig")
	crdManPath := flag.String("crd", "", "path to crd manifest")
	opManPath := flag.String("op", "", "path to operator manifest")
	rbacManPath := flag.String("rbac", "", "path to rbac manifest")
	flag.Parse()
	// go test always runs from the test directory; change to project root
	err := os.Chdir(*projRoot)
	if err != nil {
		log.Fatalf("failed to change directory to project root: %v", err)
	}
	if err := setup(kubeconfigPath, crdManPath, opManPath, rbacManPath); err != nil {
		log.Fatalf("failed to set up framework: %v", err)
	}
	// setup context to use when setting up crd
	ctx := NewTestCtx(nil)
	// os.Exit stops the program before the deferred functions run
	// to fix this, we put the exit in the defer as well
	defer func() {
		exitCode := m.Run()
		ctx.CleanupNoT()
		os.Exit(exitCode)
	}()
	// create crd
	crdYAML, err := ioutil.ReadFile(*Global.CrdManPath)
	if err != nil {
		log.Fatalf("failed to read crd file: %v", err)
	}
	err = ctx.CreateFromYAML(crdYAML)
	if err != nil {
		log.Fatalf("failed to create crd resource: %v", err)
	}
}
