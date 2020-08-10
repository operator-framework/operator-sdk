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
	"context"
	"errors"
	"time"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

func NewCmd() *cobra.Command {
	var timeout time.Duration
	cfg := &operator.Configuration{}
	cmd := &cobra.Command{
		Use:   "cleanup <operatorPackageName>",
		Short: "Clean up an Operator deployed with the 'run' subcommand",
		Long:  "This command has subcommands that will destroy an Operator deployed with OLM.",
		Args:  cobra.MaximumNArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return cfg.Load()
		},
		Run: func(cmd *cobra.Command, args []string) {
			u := operator.NewUninstall(cfg)

			// If there is no `packageName` command line argument, attempt to
			// use `projectName` from the PROJECT file as the package name.
			if len(args) == 0 {
				projectCfg, err := projutil.ReadConfig()
				if err != nil {
					log.Fatalf("Error reading configuration. If you are NOT in the root directory of the project "+
						"then, you must inform the project name: %v", err)
				}
				u.Package = projectCfg.ProjectName
			} else {
				u.Package = args[0]
			}

			u.DeleteAll = true
			u.DeleteOperatorGroupNames = []string{operator.SDKOperatorGroupName}
			u.Logf = log.Infof

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			err := u.Run(ctx)
			var pkgErr *operator.ErrPackageNotFound
			switch {
			case errors.As(err, &pkgErr):
				log.Warnf("Cleanup operator: %v\n", pkgErr)
			case err != nil:
				log.Fatalf("Cleanup operator: %v\n", err)
			default:
				log.Infof("Operator %q uninstalled\n", u.Package)
			}
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Time to wait for the command to complete before failing")
	cfg.BindFlags(cmd.PersistentFlags())

	return cmd
}
