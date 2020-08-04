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

package bundle

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/operator"
	"github.com/operator-framework/operator-sdk/internal/operator/bundle"
)

func NewCmd() *cobra.Command {
	var timeout time.Duration

	// TODO(joelanford): the initialization of cfg up to
	//   the "run" subcommand when migrating packagemanifests
	//   to this design.
	cfg := &operator.Configuration{}

	u := bundle.NewUninstall(cfg)
	cmd := &cobra.Command{
		Use:    "bundle <bundle-image>",
		Short:  "Cleanup an Operator in the bundle format with OLM",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return cfg.Load()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			u.BundleImage = args[0]

			if err := u.Run(ctx); err != nil {
				fmt.Printf("cleanup error: %v\n", err)
			}
			fmt.Printf("operator uninstalled\n")
		},
	}
	cmd.Flags().SortFlags = false
	cfg.BindFlags(cmd.PersistentFlags())
	u.BindFlags(cmd.Flags())

	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "cleanup timeout")
	return cmd
}
