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

package main

import (
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that `run` and `up local` can make use of them.
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/add"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/build"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/migrate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/new"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/olmcatalog"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/printdeps"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/run"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/test"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/up"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/version"
	flags "github.com/operator-framework/operator-sdk/internal/pkg/flags"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	root := &cobra.Command{
		Use:   "operator-sdk",
		Short: "An SDK for building operators with ease",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if viper.GetBool(flags.VerboseOpt) {
				err := projutil.SetGoVerbose()
				if err != nil {
					log.Errorf("Could not set GOFLAGS: (%v)", err)
					return
				}
				log.SetLevel(log.DebugLevel)
				log.Debug("Debug logging is set")
			}
		},
	}

	root.AddCommand(new.NewCmd())
	root.AddCommand(add.NewCmd())
	root.AddCommand(build.NewCmd())
	root.AddCommand(generate.NewCmd())
	root.AddCommand(up.NewCmd())
	root.AddCommand(completion.NewCmd())
	root.AddCommand(test.NewCmd())
	root.AddCommand(scorecard.NewCmd())
	root.AddCommand(printdeps.NewCmd())
	root.AddCommand(migrate.NewCmd())
	root.AddCommand(run.NewCmd())
	root.AddCommand(olmcatalog.NewCmd())
	root.AddCommand(version.NewCmd())

	root.PersistentFlags().Bool(flags.VerboseOpt, false, "Enable verbose logging")
	if err := viper.BindPFlags(root.PersistentFlags()); err != nil {
		log.Fatalf("Failed to bind root flags: %v", err)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
