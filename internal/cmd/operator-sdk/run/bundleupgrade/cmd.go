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

package bundleupgrade

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/bundleupgrade"
)

func NewCmd(cfg *operator.Configuration) *cobra.Command {
	u := bundleupgrade.NewUpgrade(cfg)
	cmd := &cobra.Command{
		Use:   "bundle-upgrade <bundle-image>",
		Short: "Upgrade an Operator previously installed in the bundle format with OLM",
		Long: `The single argument to this command is a bundle image, with the full registry path specified.
If using a docker.io image, you must specify docker.io(/<namespace>)?/<bundle-image-name>:<tag>.`,
		Args:    cobra.ExactArgs(1),
		PreRunE: func(*cobra.Command, []string) error { return cfg.Load() },
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
			defer cancel()

			u.BundleImage = args[0]

			_, err := u.Run(ctx)
			if err != nil {
				logrus.Fatalf("Failed to run bundle upgrade: %v\n", err)
			}
		},
	}

	cfg.BindFlags(cmd.Flags())
	u.BindFlags(cmd.Flags())

	return cmd
}
