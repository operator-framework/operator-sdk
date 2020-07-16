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

package cli

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/add"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/build"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/new"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/version"
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

func RunLegacy() error {
	root := GetCLIRoot()
	root.PersistentFlags().Bool(flags.VerboseOpt, false, "Enable verbose logging")
	if err := viper.BindPFlags(root.PersistentFlags()); err != nil {
		log.Fatalf("Failed to bind root flags: %v", err)
	}

	return root.Execute()
}

// GetCLIRoot is intended to create the base command structure for the OSDK for use in CLI and documentation
// Deprecated: this is only used for projects without a PROJECT config/kubebuilder-style repo.
func GetCLIRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "operator-sdk",
		Short: "An SDK for building operators with ease",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if viper.GetBool(flags.VerboseOpt) {
				if err := projutil.SetGoVerbose(); err != nil {
					log.Fatalf("Could not set GOFLAGS: (%v)", err)
				}
				log.SetLevel(log.DebugLevel)
				log.Debug("Debug logging is set")
			}
			if err := checkGoModulesForCmd(cmd); err != nil {
				log.Fatal(err)
			}
		},
	}

	root.AddCommand(
		add.NewCmd(),
		build.NewCmd(),
		scorecard.NewCmd(),
		completion.NewCmd(),
		new.NewCmd(),
		olm.NewCmd(),
		version.NewCmd(),
	)

	return root
}

func checkGoModulesForCmd(cmd *cobra.Command) (err error) {
	// Certain commands are able to be run anywhere or handle this check
	// differently in their CLI code.
	if skipCheckForCmd(cmd) {
		return nil
	}
	// Do not perform this check if the project is non-Go, as they will not
	// be using go modules.
	if !projutil.IsOperatorGo() {
		return nil
	}
	// Do not perform a go modules check if the working directory is not in
	// the project root, as some sub-commands might not require project root.
	// Individual subcommands will perform this check as needed.
	if err := projutil.CheckProjectRoot(); err != nil {
		return nil
	}

	return projutil.CheckGoModules()
}

var commandsToSkip = map[string]struct{}{
	"new":          struct{}{}, // Handles this logic in cmd/operator-sdk/new
	"operator-sdk": struct{}{}, // Alias for "help"
	"help":         struct{}{},
	"completion":   struct{}{},
	"version":      struct{}{},
}

func skipCheckForCmd(cmd *cobra.Command) (skip bool) {
	if _, ok := commandsToSkip[cmd.Name()]; ok {
		return true
	}
	cmd.VisitParents(func(pc *cobra.Command) {
		if _, ok := commandsToSkip[pc.Name()]; ok {
			// The bare "operator-sdk" command will be checked above.
			if pc.Name() != "operator-sdk" {
				skip = true
			}
		}
	})
	return skip
}
