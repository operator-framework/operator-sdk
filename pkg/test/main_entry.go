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
	ProjRootFlag          = "root"
	KubeConfigFlag        = "kubeconfig"
	NamespacedManPathFlag = "namespacedMan"
	GlobalManPathFlag     = "globalMan"
	SingleNamespaceFlag   = "singleNamespace"
	TestNamespaceEnv      = "TEST_NAMESPACE"
)

func MainEntry(m *testing.M) {
	projRoot := flag.String(ProjRootFlag, "", "path to project root")
	kubeconfigPath := flag.String(KubeConfigFlag, "", "path to kubeconfig")
	globalManPath := flag.String(GlobalManPathFlag, "", "path to operator manifest")
	namespacedManPath := flag.String(NamespacedManPathFlag, "", "path to rbac manifest")
	singleNamespace = flag.Bool(SingleNamespaceFlag, false, "enable single namespace mode")
	flag.Parse()
	// go test always runs from the test directory; change to project root
	err := os.Chdir(*projRoot)
	if err != nil {
		log.Fatalf("failed to change directory to project root: %v", err)
	}
	if err := setup(kubeconfigPath, namespacedManPath); err != nil {
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
	if *kubeconfigPath != "incluster" {
		globalYAML, err := ioutil.ReadFile(*globalManPath)
		if err != nil {
			log.Fatalf("failed to read global resource manifest: %v", err)
		}
		err = ctx.createFromYAML(globalYAML, true, &CleanupOptions{TestContext: ctx})
		if err != nil {
			log.Fatalf("failed to create resource(s) in global resource manifest: %v", err)
		}
	}
}
