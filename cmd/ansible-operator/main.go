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

package main

import (
	"log"

	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/operator-framework/operator-sdk/internal/cmd/ansible-operator/run"
	"github.com/operator-framework/operator-sdk/internal/cmd/ansible-operator/version"
)

func main() {
	root := cobra.Command{
		Short: "Reconcile an Ansible operator project using ansible-runner",
		Long: `This binary runs an Ansible operator that reconciles Kubernetes resources
managed by the ansible-runner program. It can be run either directly or from an Ansible
operator project's image entrypoint
`,
		Use: "ansible-operator",
	}

	root.AddCommand(run.NewCmd())
	root.AddCommand(version.NewCmd())

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
