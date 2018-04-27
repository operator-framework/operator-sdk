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

package generate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"

	"github.com/spf13/cobra"
)

const (
	k8sGenerated = "./tmp/codegen/update-generated.sh"
	// dot represents current dir.
	dot = "."
)

func NewGenerateK8SCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "k8s",
		Short: "Generates Kubernetes code for custom resource",
		Long: `k8s generator generates code for custom resource given the API spec
to comply with kube-API requirements.
`,
		Run: k8sFunc,
	}
}

func k8sFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("k8s command doesn't accept any arguments."))
	}
	K8sCodegen(dot)
}

// K8sCodegen performs code-generation for custom resources of this project given the projectDir.
func K8sCodegen(projectDir string) {
	fmt.Fprintln(os.Stdout, "Run code-generation for custom resources")
	kcmd := exec.Command(k8sGenerated)
	kcmd.Dir = projectDir
	o, err := kcmd.CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to perform code-generation for CustomResources: (%v)", string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))
}
