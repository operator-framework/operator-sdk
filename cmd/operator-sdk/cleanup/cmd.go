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

package cleanup

import (
	"path/filepath"

	olmcatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type cleanupCmd struct {
	// Common options.
	kubeconfig string
	namespace  string

	// Run type.
	olm bool

	// Run type-specific options.
	olmArgs olmoperator.OLMCmd
}

func NewCmd() *cobra.Command {
	c := &cleanupCmd{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete and clean up after a running Operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			projutil.MustInProjectRoot()

			switch {
			case c.olm:
				c.olmArgs.KubeconfigPath = c.kubeconfig
				c.olmArgs.OperatorNamespace = c.namespace
				if c.olmArgs.ManifestsDir == "" {
					operatorName := filepath.Base(projutil.MustGetwd())
					c.olmArgs.ManifestsDir = filepath.Join(olmcatalog.OLMCatalogDir, operatorName)
				}
				if err := c.olmArgs.Cleanup(); err != nil {
					log.Fatalf("Failed to clean up operator using OLM: %v", err)
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
	cmd.Flags().StringVar(&c.namespace, "namespace", "",
		"The namespace where the operator watches for changes.")

	// 'run --olm' and related flags.
	cmd.Flags().BoolVar(&c.olm, "olm", true,
		"The operator to be run will be managed by OLM in a cluster.")
	c.olmArgs.AddToFlagSet(cmd.Flags())
	return cmd
}
