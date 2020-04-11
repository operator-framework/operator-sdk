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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	migrationDocLinkGo = "https://sdk.operatorframework.io/docs/golang/project_migration_guide"
	// TODO: add ansible and helm migration doc links when ready.
)

type migrateCmd struct {
	pluginType string
	fromDir    string
	toDir      string
	license    string
	owner      string
	repo       string

	verbose bool
}

func rootCmd() *cobra.Command {
	c := &migrateCmd{}

	cmd := &cobra.Command{
		Use:   "operator-migrate",
		Short: "migrates a legacy Operator SDK project to a Kubebuilder-style project",
		Long: `Migrate a legacy Operator SDK project to a Kubebuilder-style project
by scaffolding a new project using 'operator-sdk init' and adding APIs using
'operator-sdk create api', both with current project data.
`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if c.verbose {
				if err := projutil.SetGoVerbose(); err != nil {
					log.Fatalf("Could not set GOFLAGS: (%v)", err)
				}
				log.SetLevel(log.DebugLevel)
				log.Debug("Debug logging is set")
			}
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			projectConfig := filepath.Join(c.fromDir, "PROJECT")
			if _, err = os.Stat(projectConfig); err == nil || os.IsExist(err) {
				log.Fatalf("Project %s is already migrated", c.fromDir)
			}

			var layout pluginKey
			if c.pluginType != "" {
				if layout, err = toPluginKey(c.pluginType); err != nil {
					return err
				}
			}

			if projutil.CheckProjectRoot() != nil {
				if c.fromDir == "" {
					return fmt.Errorf("--from must be set if not running in a project")
				}
				if c.pluginType == "" {
					return fmt.Errorf("--type must be set if not running in a project")
				}
			} else {
				if c.fromDir == "" {
					c.fromDir = "."
				}
				if c.pluginType == "" {
					switch t := projutil.GetOperatorType(); t {
					case projutil.OperatorTypeGo:
						layout = pluginKeyGo
					default:
						log.Fatalf(`Migration is not supported for project type %s, possible values: ["go"]`, t)
					}
				}
			}

			if err = c.runWithPlugin(layout); err != nil && !errors.As(err, &needsHelpErr{}) {
				log.Fatal(err)
			}
			return err
		},
	}

	cmd.PersistentFlags().BoolVar(&c.verbose, flags.VerboseOpt, false, "enable verbose logging")

	cmd.Flags().StringVar(&c.pluginType, "type", "", `type of project being migrated, possible values: ["go"]`)
	cmd.Flags().StringVar(&c.license, "license", "", "license type to use")
	cmd.Flags().StringVar(&c.repo, "repo", "",
		"project repository path or name. If the project is a Go operator, the path "+
			"must be a module path (default is the legacy project's go.mod module). "+
			"Otherwise the repo is the project's name (default is the legacy project's directory name)")
	cmd.Flags().StringVar(&c.owner, "owner", "", "owner of the project, this information goes into the license")
	cmd.Flags().StringVar(&c.fromDir, "from-dir", "", "directory of project to migrate")
	cmd.Flags().StringVar(&c.toDir, "to-dir", "", "directory to place migrated project")

	return cmd
}

type needsHelpErr struct {
	err error
}

func (e needsHelpErr) Error() string {
	return e.err.Error()
}

func (e needsHelpErr) Unwrap() error {
	return e.err
}
