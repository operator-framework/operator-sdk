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

package packagemanifests

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/packagemanifests"
)

func NewCmd(cfg *operator.Configuration) *cobra.Command {
	i := packagemanifests.NewInstall(cfg)
	cmd := &cobra.Command{
		Use: "packagemanifests [packagemanifests-root-dir]",
		Deprecated: "support for the packagemanifests format will be removed in operator-sdk v2.0.0. Use bundles to " +
			"package your operator instead. Migrate your packagemanifest to a bundle using " +
			"'operator-sdk pkgman-to-bundle' command. Run 'operator-sdk pkgman-to-bundle --help' " +
			"for more details.",
		Short: "Deploy an Operator in the package manifests format with OLM",
		Long: `'run packagemanifests' deploys an Operator's package manifests with OLM. The command's argument
will default to './packagemanifests' if unset; if set, the argument must be a package manifests root directory,
ex. '<project-root>/packagemanifests'.`,
		Aliases: []string{"pm"},
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(*cobra.Command, []string) error { return cfg.Load() },
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
			defer cancel()

			if len(args) == 0 {
				i.PackageManifestsDirectory = "packagemanifests"
			} else {
				i.PackageManifestsDirectory = args[0]
			}

			// TODO(joelanford): Add cleanup logic if this fails?
			_, err := i.Run(ctx)
			if err != nil {
				log.Fatalf("Failed to run packagemanifests: %v\n", err)
			}
		},
	}

	cfg.BindFlags(cmd.Flags())
	i.BindFlags(cmd.Flags())
	// Not implemented.
	if err := cmd.Flags().MarkHidden("service-account"); err != nil {
		log.Fatal(err)
	}

	return cmd
}
