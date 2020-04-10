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
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Use new commands if in a configured project or not in a project.
	if kbutil.IsConfigExist() || projutil.CheckProjectRoot() != nil {
		if err := cli.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := cli.RunLegacy(); err != nil {
		os.Exit(1)
	}
}
