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

	"github.com/operator-framework/operator-sdk/internal/util/projutil"

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

func checkDepManagerForCmd(cmd *cobra.Command) (err error) {
	// Do not perform this check if wd is not in the project root,
	// as some sub-commands might not require project root.
	if err := projutil.CheckProjectRoot(); err != nil {
		return nil
	}

	var dm projutil.DepManagerType
	switch cmd.Name() {
	case "new", "migrate":
		// Do not perform this check if the new project is non-Go, as they do not
		// have (Go) dep managers.
		if cmd.Name() == "new" {
			projType, err := cmd.Flags().GetString("type")
			if err != nil {
				return err
			}
			if projType != "go" {
				return nil
			}
		}
		// "new" and "migrate" commands are for projects that establish which
		// dep manager to use. "new" should not be called if we're in the project
		// root but could be, so check it anyway.
		dmStr, err := cmd.Flags().GetString("dep-manager")
		if err != nil {
			return err
		}
		dm = projutil.DepManagerType(dmStr)
	default:
		// Do not perform this check if the project is non-Go, as they do not
		// have (Go) dep managers.
		if !projutil.IsOperatorGo() {
			return nil
		}
		if dm, err = projutil.GetDepManagerType(); err != nil {
			return err
		}
	}

	switch dm {
	case projutil.DepManagerGoMod:
		goModOn, err := projutil.GoModOn()
		if err != nil {
			return err
		}
		if !goModOn {
			return fmt.Errorf(`depedency manger "modules" requires wd be in $GOPATH/src` +
				` and GO111MODULE=on, or outside of $GOPATH/src and GO111MODULE="on", "auto", or unset`)
		}
	case projutil.DepManagerDep:
		inGopathSrc, err := projutil.WdInGoPathSrc()
		if err != nil {
			return err
		}
		if !inGopathSrc {
			return fmt.Errorf(`depedency manger "dep" requires wd be in $GOPATH/src`)
		}
	}

	return nil
}
