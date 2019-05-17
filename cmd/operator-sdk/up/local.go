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

package up

import (
	"fmt"

	k8sinternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/up"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// newLocalCmd - up local command to run an operator loccally
func newLocalCmd() *cobra.Command {
	c := &up.UpLocalCmd{}
	aoFlags := &aoflags.AnsibleOperatorFlags{}
	hoFlags := &hoflags.HelmOperatorFlags{}

	upLocalCmd := &cobra.Command{
		Use:   "local",
		Short: "Launches the operator locally",
		Long: `The operator-sdk up local command launches the operator on the local machine
by building the operator binary with the ability to access a
kubernetes cluster using a kubeconfig file.
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			t := projutil.GetOperatorType()
			if t == projutil.OperatorTypeUnknown {
				return projutil.ErrUnknownOperatorType{}
			}

			log.Info("Running the operator locally.")

			// get default namespace to watch if unset
			if !cmd.Flags().Changed("namespace") {
				_, c.Namespace, err = k8sinternal.GetKubeconfigAndNamespace(c.KubeconfigPath)
				if err != nil {
					return fmt.Errorf("failed to get kubeconfig and default namespace: %v", err)
				}
			}
			log.Infof("Using namespace %s.", c.Namespace)

			if t == projutil.OperatorTypeGo {
				return c.Run()
			}
			logf.SetLogger(zap.Logger())
			if err := up.SetupOperatorEnv(c.KubeconfigPath, c.Namespace); err != nil {
				return err
			}
			switch t {
			case projutil.OperatorTypeAnsible:
				return ansible.Run(aoFlags)
			case projutil.OperatorTypeHelm:
				return helm.Run(hoFlags)
			}
			return nil
		},
	}

	upLocalCmd.Flags().StringVar(&c.KubeconfigPath, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to location specified by $KUBECONFIG with a fallback to $HOME/.kube/config if not set")
	upLocalCmd.Flags().StringVar(&c.OperatorFlags, "operator-flags", "", "The flags that the operator needs. Example: \"--flag1 value1 --flag2=value2\"")
	upLocalCmd.Flags().StringVar(&c.Namespace, "namespace", "", "The namespace where the operator watches for changes.")
	upLocalCmd.Flags().StringVar(&c.LDFlags, "go-ldflags", "", "Set Go linker options")
	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeAnsible:
		aoFlags = aoflags.AddTo(upLocalCmd.Flags(), "(ansible operator)")
	case projutil.OperatorTypeHelm:
		hoFlags = hoflags.AddTo(upLocalCmd.Flags(), "(helm operator)")
	}
	return upLocalCmd
}
