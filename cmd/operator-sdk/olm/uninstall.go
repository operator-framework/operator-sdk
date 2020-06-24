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

package olm

import (
	"github.com/operator-framework/operator-sdk/internal/olm"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	mgr := olm.Manager{}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Operator Lifecycle Manager from your cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgr.Uninstall(); err != nil {
				log.Fatalf("Failed to uninstall OLM: %s", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mgr.Version, "version", "", "version of OLM resources to uninstall.")
	mgr.AddToFlagSet(cmd.Flags())
	return cmd
}
