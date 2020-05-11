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
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that `exec-entrypoint` and `run` can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cli"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Use the new KB CLI only when running inside an existing Kubebuilder project with a PROJECT file.
	// The default legacy CLI provides the init cmd to initialize
	// a Kubebuilder project as a way to opt into the new KB CLI.
	// TODO: Make the new KB CLI the default, once the integration is complete
	// and deprecate "operator-sdk new" from the old CLI.
	if kbutil.HasProjectFile() {
		if err := cli.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := cli.RunLegacy(); err != nil {
		os.Exit(1)
	}
}
