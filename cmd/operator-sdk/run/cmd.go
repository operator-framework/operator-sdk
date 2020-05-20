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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/run/local"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/run/packagemanifests"
	olmcatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	k8sinternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
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
	olmArgs   olmoperator.PackageManifestsCmd
	localArgs local.RunLocalCmd
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
		Long: `This command has subcommands that will run or deploy your Operator in two
different modes: locally and using OLM. These modes are controlled by using 'local'
or 'packagemanifests' subcommands. Run 'operator-sdk run --help' for more
information on these subcommands.
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
				// The main.go and manager.yaml scaffolds in the new layout do not support the WATCH_NAMESPACE
				// env var to configure the namespace that the operator watches. The default is all namespaces.
				// So this flag is unsupported for the new layout.
				if !kbutil.HasProjectFile() {
					//TODO: remove namespace flag before 1.0.0
					// set --watch-namespace flag if the --namespace flag is set
					// (only if --watch-namespace flag is not set)
					if cmd.Flags().Changed("namespace") { // not valid for te new layout
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
						c.localArgs.WatchNamespace = defaultNamespace
					}
				}

				c.localArgs.Kubeconfig = c.kubeconfig
				if err := c.localArgs.Run(); err != nil {
					log.Fatalf("Failed to run operator locally: %v", err)
				}
			}
			return nil
		},
	}

	// Shared flags.
	cmd.Flags().StringVar(&c.kubeconfig, "kubeconfig", "",
		"The file path to kubernetes configuration file. Defaults to location "+
			"specified by $KUBECONFIG, or to default file rules if not set")
	err := cmd.Flags().MarkDeprecated("kubeconfig",
		"use --kubeconfig with 'local' or 'packagemanifests' subcommands instead")
	if err != nil {
		panic(err)
	}
	// Deprecated: namespace exists for historical compatibility. Use watch-namespace instead.
	//TODO: remove namespace flag before 1.0.0
	if !kbutil.HasProjectFile() { // not show for the kb layout projects
		cmd.Flags().StringVar(&c.namespace, "namespace", "",
			"The namespace in which operator and namespaces resources are run")
		err := cmd.Flags().MarkDeprecated("namespace", "use --watch-namespaces (with --local) "+
			"or --operator-namespace (with --olm) instead")
		if err != nil {
			panic(err)
		}
	}

	// 'run --olm' and related flags.
	cmd.Flags().BoolVar(&c.olm, "olm", false,
		"The operator to be run will be managed by OLM in a cluster. "+
			"Cannot be set with another run-type flag")
	err = cmd.Flags().MarkDeprecated("olm", "use 'run packagemanifests' instead")
	if err != nil {
		panic(err)
	}
	// Mark all flags used with '--olm' as deprecated and hidden separately so
	// all other 'run' flags are still available.
	olmFS := pflag.NewFlagSet("olm", pflag.ExitOnError)
	olmFS.StringVar(&c.olmArgs.ManifestsDir, "manifests", "",
		"Directory containing operator package directories and a package manifest file")
	c.olmArgs.AddToFlagSet(olmFS)
	olmFS.VisitAll(func(f *pflag.Flag) {
		f.Deprecated = "use this flag with 'run packagemanifests' instead"
		f.Hidden = true
	})
	cmd.Flags().AddFlagSet(olmFS)

	// 'run --local' and related flags.
	cmd.Flags().BoolVar(&c.local, "local", false,
		"The operator will be run locally by building the operator binary with "+
			"the ability to access a kubernetes cluster using a kubeconfig file. "+
			"Cannot be set with another run-type flag.")
	err = cmd.Flags().MarkDeprecated("local", "use 'run local' instead")
	if err != nil {
		panic(err)
	}
	localFS := pflag.NewFlagSet("local", pflag.ExitOnError)
	c.localArgs.AddToFlags(localFS)
	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeAnsible:
		c.localArgs.AnsibleOperatorFlags = aoflags.AddTo(localFS, "(ansible operator)")
	case projutil.OperatorTypeHelm:
		c.localArgs.HelmOperatorFlags = hoflags.AddTo(localFS, "(helm operator)")
	}
	localFS.VisitAll(func(f *pflag.Flag) {
		f.Deprecated = "use this flag with 'run local' instead"
		f.Hidden = true
	})
	cmd.Flags().AddFlagSet(localFS)

	cmd.AddCommand(
		packagemanifests.NewCmd(),
		local.NewCmd(),
	)

	return cmd
}
