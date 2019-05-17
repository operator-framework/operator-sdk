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
	"os"
	"path/filepath"
	"strings"

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
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/writeconfig"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"
	osdkversion "github.com/operator-framework/operator-sdk/version"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that `run` and `up local` can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// These commands do not need a config file.
var skipConfigCmds = map[string]struct{}{
	"new":          struct{}{},
	"write-config": struct{}{},
}

func main() {
	var cfgFile string

	root := &cobra.Command{
		Use:     "operator-sdk",
		Short:   "An SDK for building operators with ease",
		Version: osdkversion.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetDefaults(); err != nil {
				return err
			}

			if _, ok := skipConfigCmds[cmd.Name()]; ok {
				return nil
			}

			if cfgFile != "" {
				if _, err := os.Stat(cfgFile); err != nil {
					return errors.Wrapf(err, "error getting config file %s", cfgFile)
				}
				viper.SetConfigFile(cfgFile)
			} else {
				path := projutil.MustGetwd()
				viper.AddConfigPath(path)
				fn := config.DefaultFileName
				viper.SetConfigName(strings.TrimSuffix(fn, filepath.Ext(fn)))
			}

			if err := viper.ReadInConfig(); err == nil {
				log.Infof("Using config file %s", viper.ConfigFileUsed())
			} else {
				log.Info("No config file found. Using defaults and flags.")
			}

			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if viper.GetBool(config.VerboseOpt) {
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
	root.AddCommand(test.NewCmd())
	root.AddCommand(scorecard.NewCmd())
	root.AddCommand(printdeps.NewCmd())
	root.AddCommand(migrate.NewCmd())
	root.AddCommand(run.NewCmd())
	root.AddCommand(olmcatalog.NewCmd())
	root.AddCommand(completion.NewCmd())
	root.AddCommand(version.NewCmd())
	root.AddCommand(writeconfig.NewCmd())

	root.PersistentFlags().Bool(config.VerboseOpt, false, "Enable verbose logging")
	viper.BindPFlag(config.VerboseOpt, root.PersistentFlags().Lookup(config.VerboseOpt))
	root.PersistentFlags().StringVar(&cfgFile, config.ConfigOpt, "", "Operator SDK global configuration file")

	// Ensure persistent flags are inherited by all children.
	cmds := root.Commands()
	numCmds := len(cmds)
	for numCmds > 0 {
		for _, cmd := range cmds {
			cmd.LocalFlags()
			cmds = append(cmds, cmd.Commands()...)
		}
		cmds = cmds[numCmds:]
		numCmds = len(cmds) - numCmds
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
