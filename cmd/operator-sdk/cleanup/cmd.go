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
	"errors"
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

	// Cleanup type.
	olm bool

	// Cleanup type-specific options.
	olmArgs olmoperator.OLMCmd
}

// checkCleanupType ensures exactly one cleanup type has been selected.
func (c *cleanupCmd) checkCleanupType() error {
	if !c.olm {
		return errors.New("exactly one run-type flag must be set: --olm")
	}
	return nil
}

func NewCmd() *cobra.Command {
	c := &cleanupCmd{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete and clean up after a running Operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.checkCleanupType(); err != nil {
				return err
			}
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

	// Avoid sorting flags so we can group them according to cleanup type.
	cmd.Flags().SortFlags = false

	// Shared flags.
	cmd.Flags().StringVar(&c.kubeconfig, "kubeconfig", "",
		"The file path to kubernetes configuration file. Defaults to location "+
			"specified by $KUBECONFIG, or to default file rules if not set")
	cmd.Flags().StringVar(&c.namespace, "namespace", "",
		"The namespace where the operator watches for changes.")

	// 'cleanup --olm' and related flags. Set as default since this is the only
	// cleanup type.
	cmd.Flags().BoolVar(&c.olm, "olm", true,
		"The operator to be cleaned up is managed by OLM in a cluster. "+
			"Cannot be set with another cleanup-type flag")
	c.olmArgs.AddToFlagSet(cmd.Flags())
	return cmd
}
