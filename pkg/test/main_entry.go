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
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func MainEntry(m *testing.M) {
	fopts := &frameworkOpts{}
	fopts.addToFlagSet(flag.CommandLine)
	// controller-runtime registers the --kubeconfig flag in client config
	// package:
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.5.2/pkg/client/config/config.go#L39
	//
	// If this flag is not registered, do so. Otherwise retrieve its value.
	kcFlag := flag.Lookup(KubeConfigFlag)
	if kcFlag == nil {
		flag.StringVar(&fopts.kubeconfigPath, KubeConfigFlag, "", "path to kubeconfig")
	}

	flag.Parse()

	if kcFlag != nil {
		fopts.kubeconfigPath = kcFlag.Value.String()
	}

	f, err := newFramework(fopts)
	if err != nil {
		log.Fatalf("Failed to create framework: %v", err)
	}

	Global = f

	exitCode, err := f.runM(m)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(exitCode)
}
