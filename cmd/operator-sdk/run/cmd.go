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

	olmcatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
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
	//TODO: remove namespace flag before 1.0.0
	//namespace is deprecated
	namespace string

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
		Long: `This command will run or deploy your Operator in two different modes: locally
and using OLM. These modes are controlled by setting --local and --olm run mode
flags. Each run mode has a separate set of flags that configure 'run' for that
mode. Run 'operator-sdk run --help' for more information on these flags.

Read more about the --olm run mode and configuration options here:
https://github.com/operator-framework/operator-sdk/blob/master/doc/user/olm-catalog/olm-cli.md
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.checkRunType(); err != nil {
				return err
			}
			projutil.MustInProjectRoot()

			switch {
			case c.olm:
				c.olmArgs.KubeconfigPath = c.kubeconfig
				// operator-namespace flag is not set
				// use default namespace from kubeconfig to deploy operator resources
				if !cmd.Flags().Changed("operator-namespace") {
					_, defaultNamespace, err := k8sinternal.GetKubeconfigAndNamespace(c.kubeconfig)
					if err != nil {
						return fmt.Errorf("error getting kubeconfig and default namespace: %v", err)
					}
					c.olmArgs.OperatorNamespace = defaultNamespace
				}
				if c.olmArgs.ManifestsDir == "" {
					operatorName := filepath.Base(projutil.MustGetwd())
					c.olmArgs.ManifestsDir = filepath.Join(olmcatalog.OLMCatalogDir, operatorName)
				}
				if err := c.olmArgs.Run(); err != nil {
					log.Fatalf("Failed to run operator using OLM: %v", err)
				}
			case c.local:
				//TODO: remove namespace flag before 1.0.0
				// set --watch-namespace flag if the --namespace flag is set
				// (only if --watch-namespace flag is not set)
				if cmd.Flags().Changed("namespace") {
					log.Info("--namespace is deprecated; use --watch-namespace instead.")
					if !cmd.Flags().Changed("watch-namespace") {
						err := cmd.Flags().Set("watch-namespace", c.namespace)
						return err
					}
				}
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
	// Deprecated: namespace exists for historical compatibility. Use watch-namespace instead.
	//TODO: remove namespace flag before 1.0.0
	cmd.Flags().StringVar(&c.namespace, "namespace", "",
		"(Deprecated: use --watch-namespace instead.)"+
			"The namespace where the operator watches for changes.")
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
