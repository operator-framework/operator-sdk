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
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/operator"
	"github.com/operator-framework/operator-sdk/internal/operator/bundle"
)

func NewCmd() *cobra.Command {
	var timeout time.Duration
	var cleanupTimeout time.Duration

	// TODO(joelanford): the initialization of cfg up to
	//   the "run" subcommand when migrating packagemanifests
	//   to this design.
	cfg := &operator.Configuration{}

	i := bundle.NewInstall(cfg)
	u := bundle.NewUninstall(cfg)
	cmd := &cobra.Command{
		Use:    "bundle <bundle-image>",
		Short:  "Deploy an Operator in the bundle format with OLM",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return cfg.Load()
		},
		Run: func(cmd *cobra.Command, args []string) {
			runCtx, runCancel := context.WithTimeout(cmd.Context(), timeout)
			defer runCancel()

			i.BundleImage = args[0]
			u.BundleImage = args[0]
			u.DeleteAll = true

			csv, err := i.Run(runCtx)
			if err != nil {
				func() {
					cancelCtx, cancelCancel := context.WithTimeout(cmd.Context(), cleanupTimeout)
					defer cancelCancel()

					cleanupErr := u.Run(cancelCtx)
					if cleanupErr != nil {
						defer func() {
							fmt.Printf("cleanup error: %v\n", cleanupErr)
						}()
					}
					fmt.Printf("failed to run bundle: %v\n", err)
				}()
				os.Exit(1)
			}
			fmt.Printf("csv %q installed\n", csv.Name)
		},
	}
	cmd.Flags().SortFlags = false
	cfg.BindFlags(cmd.PersistentFlags())
	i.BindFlags(cmd.Flags())

	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "install timeout")
	cmd.Flags().DurationVar(&cleanupTimeout, "cleanup-timeout", 10*time.Second, "cleanup timeout")
	return cmd
}
