// Copyright 2019 The Operator-SDK Authors
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

package main

import (

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that `exec-entrypoint` and `run` can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cli"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Use the new KB CLI when running inside a Kubebuilder project with an existing PROJECT file.
	if kbutil.HasProjectFile() {
		if err := cli.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Use the legacy CLI if inside of a Go/Helm/Ansible legacy project
	operatorType := projutil.GetOperatorType()
	switch operatorType {
	case projutil.OperatorTypeGo, projutil.OperatorTypeHelm, projutil.OperatorTypeAnsible:
		// Deprecation warning for Go projects
		// TODO/Discuss: UX wise, is displaying this notice on every command that runs
		// in the legacy Go projects too loud.
		if operatorType == projutil.OperatorTypeGo {
			depMsg := "Operator SDK has a new CLI and project layout that is aligned with Kubebuilder.\n" +
				"See `operator-sdk init -h` and the following doc on how to scaffold a new project:\n" +
				"https://sdk.operatorframework.io/docs/golang/quickstart/\n" +
				"To migrate existing projects to the new layout see:\n" +
				"https://sdk.operatorframework.io/docs/golang/migration/project_migration_guide/\n"
			projutil.PrintDeprecationWarning(depMsg)
		}
		if err := cli.RunLegacy(); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Run the KB CLI when not running in either legacy or new projects
	// The new CLI still supports "operator-sdk new --type=Ansible/Helm"
	if err := cli.Run(); err != nil {
		log.Fatal(err)
	}
}
