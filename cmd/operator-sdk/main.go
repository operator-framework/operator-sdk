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
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that `run` and `up local` can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/add"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/alpha"
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
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
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
			if err := checkDepManagerForCmd(cmd); err != nil {
				log.Fatal(err)
			}
		},
	}

	root.AddCommand(add.NewCmd())
	root.AddCommand(alpha.NewCmd())
	root.AddCommand(build.NewCmd())
	root.AddCommand(completion.NewCmd())
	root.AddCommand(generate.NewCmd())
	root.AddCommand(migrate.NewCmd())
	root.AddCommand(new.NewCmd())
	root.AddCommand(olmcatalog.NewCmd())
	root.AddCommand(printdeps.NewCmd())
	root.AddCommand(run.NewCmd())
	root.AddCommand(scorecard.NewCmd())
	root.AddCommand(test.NewCmd())
	root.AddCommand(up.NewCmd())
	root.AddCommand(version.NewCmd())

	root.PersistentFlags().Bool(flags.VerboseOpt, false, "Enable verbose logging")
	if err := viper.BindPFlags(root.PersistentFlags()); err != nil {
		log.Fatalf("Failed to bind root flags: %v", err)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkDepManagerForCmd(cmd *cobra.Command) (err error) {
	// Certain commands are able to be run anywhere or handle this check
	// differently in their CLI code.
	if skipCheckForCmd(cmd) {
		return nil
	}
	// Do not perform this check if the project is non-Go, as they will not have
	// a (Go) dep manager.
	if !projutil.IsOperatorGo() {
		return nil
	}
	// Do not perform a dep manager check if the working directory is not in
	// the project root, as some sub-commands might not require project root.
	// Individual subcommands will perform this check as needed.
	if err := projutil.CheckProjectRoot(); err != nil {
		return nil
	}

	dm, err := projutil.GetDepManagerType()
	if err != nil {
		return err
	}
	return checkDepManager(dm)
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

func checkDepManager(dm projutil.DepManagerType) error {
	switch dm {
	case projutil.DepManagerGoMod:
		goModOn, err := projutil.GoModOn()
		if err != nil {
			return err
		}
		if !goModOn {
			return fmt.Errorf(`dependency manager "modules" requires working directory to be in $GOPATH/src` +
				` and GO111MODULE=on, or outside of $GOPATH/src and GO111MODULE="on", "auto", or unset. More info: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#go-modules`)
		}
	case projutil.DepManagerDep:
		inGopathSrc, err := projutil.WdInGoPathSrc()
		if err != nil {
			return err
		}
		if !inGopathSrc {
			return fmt.Errorf(`dependency manager "dep" requires working directory to be in $GOPATH/src`)
		}
	default:
		return projutil.ErrInvalidDepManager(dm)
	}
	return nil
}
