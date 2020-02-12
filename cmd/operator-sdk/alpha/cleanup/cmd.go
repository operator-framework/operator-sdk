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
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type cleanupArgs struct {
	olm bool
}

func NewCmd() *cobra.Command {
	cargs := &cleanupArgs{}
	c := &olmoperator.OLMCmd{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete and clean up after a running Operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case cargs.olm:
				if err := c.Cleanup(); err != nil {
					log.Fatalf("Failed to clean up operator: %v", err)
				}
			}
			return nil
		},
	}
	// OLM is the default.
	cmd.Flags().BoolVar(&cargs.olm, "olm", true,
		"The operator to be deleted is managed by OLM in a cluster.")
	// TODO(estroz): refactor flag setting when new run mode options are added.
	c.AddToFlagSet(cmd.Flags())
	cmd.Flags().BoolVar(&c.ForceRegistry, "force-registry", false,
		"Force deletion of the in-cluster registry.")
	return cmd
}
