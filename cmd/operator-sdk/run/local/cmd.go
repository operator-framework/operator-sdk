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

package local

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	k8sinternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
)

func NewCmd() *cobra.Command {
	c := &RunLocalCmd{}

	cmd := &cobra.Command{
		Use:   "local",
		Short: "Run an Operator locally",
		Long: `This command will run your Operator locally by building the operator binary
with the ability to access a kubernetes cluster using a kubeconfig file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projutil.MustInProjectRoot()

			// The main.go and manager.yaml scaffolds in the new layout do not support the WATCH_NAMESPACE
			// env var to configure the namespace that the operator watches. The default is all namespaces.
			// So this flag is unsupported for the new layout.
			if !kbutil.HasProjectFile() {
				// Get default namespace to watch if unset.
				if !cmd.Flags().Changed("watch-namespace") {
					_, defaultNamespace, err := k8sinternal.GetKubeconfigAndNamespace(c.Kubeconfig)
					if err != nil {
						return fmt.Errorf("error getting kubeconfig and default namespace: %v", err)
					}
					c.WatchNamespace = defaultNamespace
				}
			}

			if err := c.Run(); err != nil {
				log.Fatalf("Failed to run operator: %v", err)
			}
			return nil
		},
	}

	c.AddToFlags(cmd.Flags())
	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeAnsible:
		c.AnsibleOperatorFlags = aoflags.AddTo(cmd.Flags(), "(ansible operator)")
	case projutil.OperatorTypeHelm:
		c.HelmOperatorFlags = hoflags.AddTo(cmd.Flags(), "(helm operator)")
	}

	return cmd
}
