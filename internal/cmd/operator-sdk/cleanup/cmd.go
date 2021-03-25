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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

func NewCmd() *cobra.Command {
	cfg := &operator.Configuration{}
	u := operator.NewUninstall(cfg)
	cmd := &cobra.Command{
		Use:     "cleanup <operatorPackageName>",
		Short:   "Clean up an Operator deployed with the 'run' subcommand",
		Long:    "This command has subcommands that will destroy an Operator deployed with OLM.",
		Args:    cobra.ExactArgs(1),
		PreRunE: func(*cobra.Command, []string) error { return cfg.Load() },
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			u.DeleteOperatorGroupNames = []string{operator.SDKOperatorGroupName}
			u.Logf = log.Infof

			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
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

	cfg.BindFlags(cmd.Flags())
	u.BindFlags(cmd.Flags())
	// --service-account is meaningless here.
	if err := cmd.Flags().MarkHidden("service-account"); err != nil {
		log.Fatal(err)
	}

	return cmd
}
