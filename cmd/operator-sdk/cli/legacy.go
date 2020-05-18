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
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/add"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/alpha"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/build"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/bundle"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cleanup"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/execentrypoint"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/migrate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/new"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/printdeps"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/run"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/test"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/version"
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		alpha.NewCmd(),
		build.NewCmd(),
		bundle.NewCmd(),
		cleanup.NewCmd(),
		completion.NewCmd(),
		execentrypoint.NewCmd(),
		generate.NewCmdLegacy(),
		migrate.NewCmd(),
		new.NewCmd(),
		olm.NewCmd(),
		printdeps.NewCmd(),
		run.NewCmd(),
		scorecard.NewCmd(),
		test.NewCmd(),
		version.NewCmd(),
	)

	// The "init" cmd is is added as a hidden subcommand from the new KB CLI to provide
	// a way to initialize Kubebuilder projects from the legacy CLI.
	// TODO(hasbro17): This should be removed once the KB CLI is the default.
	initCmd := getInitCmd()
	if initCmd == nil {
		log.Debug("Failed to add init cmd to CLI")
	} else {
		initCmd.Hidden = true
		root.AddCommand(initCmd)
	}

	return root
}

func getInitCmd() *cobra.Command {
	_, root := GetPluginsCLIAndRoot()
	var initCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
		}
	}
	return initCmd
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
	"migrate":      struct{}{}, // Handles this logic in cmd/operator-sdk/migrate
	"operator-sdk": struct{}{}, // Alias for "help"
	"help":         struct{}{},
	"completion":   struct{}{},
	"version":      struct{}{},
	"print-deps":   struct{}{}, // Does not require this logic
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
