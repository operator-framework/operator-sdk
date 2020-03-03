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

package run

import (
	"errors"
	"fmt"
	"path/filepath"

	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	olmcatalog "github.com/operator-framework/operator-sdk/internal/scaffold/olm-catalog"
	k8sinternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runCmd struct {
	// Common options.
	kubeconfig string

	// Run type.
	olm, local bool

	// Run type-specific options.
	olmArgs   olmoperator.OLMCmd
	localArgs runLocalArgs
}

// checkRunType ensures exactly one run type has been selected.
func (c *runCmd) checkRunType() error {
	if c.olm && c.local || !c.olm && !c.local {
		return errors.New("exactly one run-type flag must be set: --olm, --local")
	}
	return nil
}

func NewCmd() *cobra.Command {
	c := &runCmd{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an Operator in a variety of environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.checkRunType(); err != nil {
				return err
			}
			projutil.MustInProjectRoot()

			switch {
			case c.olm:
				c.olmArgs.KubeconfigPath = c.kubeconfig
				if c.olmArgs.ManifestsDir == "" {
					operatorName := filepath.Base(projutil.MustGetwd())
					c.olmArgs.ManifestsDir = filepath.Join(olmcatalog.OLMCatalogDir, operatorName)
				}
				if err := c.olmArgs.Run(); err != nil {
					log.Fatalf("Failed to run operator using OLM: %v", err)
				}
			case c.local:
				// Get default namespace to watch if unset.
				if !cmd.Flags().Changed("watch-namespace") {
					_, defaultNamespace, err := k8sinternal.GetKubeconfigAndNamespace(c.kubeconfig)
					if err != nil {
						return fmt.Errorf("error getting kubeconfig and default namespace: %v", err)
					}
					c.localArgs.watchNamespace = defaultNamespace
				}
				c.localArgs.kubeconfig = c.kubeconfig
				if err := c.localArgs.run(); err != nil {
					log.Fatalf("Failed to run operator locally: %v", err)
				}
			}
			return nil
		},
	}
	// Avoid sorting flags so we can group them according to run type.
	cmd.Flags().SortFlags = false

	// Shared flags.
	cmd.Flags().StringVar(&c.kubeconfig, "kubeconfig", "",
		"The file path to kubernetes configuration file. Defaults to location "+
			"specified by $KUBECONFIG, or to default file rules if not set")
	// 'run --olm' and related flags.
	cmd.Flags().BoolVar(&c.olm, "olm", false,
		"The operator to be run will be managed by OLM in a cluster. "+
			"Cannot be set with another run-type flag")
	c.olmArgs.AddToFlagSet(cmd.Flags())

	// 'run --local' and related flags.
	cmd.Flags().BoolVar(&c.local, "local", false,
		"The operator will be run locally by building the operator binary with "+
			"the ability to access a kubernetes cluster using a kubeconfig file. "+
			"Cannot be set with another run-type flag.")
	c.localArgs.addToFlags(cmd.Flags())
	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeAnsible:
		c.localArgs.ansibleOperatorFlags = aoflags.AddTo(cmd.Flags(), "(ansible operator)")
	case projutil.OperatorTypeHelm:
		c.localArgs.helmOperatorFlags = hoflags.AddTo(cmd.Flags(), "(helm operator)")
	}
	return cmd
}
