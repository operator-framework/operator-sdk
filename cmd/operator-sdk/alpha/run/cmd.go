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

package run

import (
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runArgs struct {
	olm bool
}

func NewCmd() *cobra.Command {
	cargs := &runArgs{}
	c := &olmoperator.OLMCmd{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an Operator in a variety of environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case cargs.olm:
				if err := c.Run(); err != nil {
					log.Fatalf("Failed to run operator: %v", err)
				}
			}
			return nil
		},
	}
	// OLM is the default.
	cmd.Flags().BoolVar(&cargs.olm, "olm", true, "The operator to be run will be managed by OLM in a cluster.")
	// TODO(estroz): refactor flag setting when new run mode options are added.
	c.AddToFlagSet(cmd.Flags())
	return cmd
}
