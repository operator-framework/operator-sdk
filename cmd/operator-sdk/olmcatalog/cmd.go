// Copyright 2018 The Operator-SDK Authors
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

package olmcatalog

import (
	"fmt"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	catalog "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	writeCSVConfigOpt = "write-csv-config"

	writeCSVConfigPath string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olm-catalog [flags] <olm-catalog-command>",
		Short: "Invokes a olm-catalog command",
		Long: `The operator-sdk olm-catalog command invokes a command to perform
Catalog related actions.`,
		RunE: olmCatalogFunc,
	}
	cmd.AddCommand(newGenCSVCmd())

	i, err := (&catalog.CSVConfig{}).GetInput()
	if err != nil {
		log.Fatalf("Error retrieving CSV config path: %v", err)
	}
	cmd.PersistentFlags().StringVar(&writeCSVConfigPath, writeCSVConfigOpt, "",
		"Write a default CSV config file. A default path is used if no value is provided."+
			" Set --write-csv-config=<path> to supply a non-default path")
	cmd.Flag(writeCSVConfigOpt).NoOptDefVal = i.Path

	return cmd
}

func olmCatalogFunc(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed(writeCSVConfigOpt) {
		absProjectPath := projutil.MustGetwd()
		cfg := &input.Config{
			AbsProjectPath: absProjectPath,
			ProjectName:    filepath.Base(absProjectPath),
		}
		if projutil.IsOperatorGo() {
			cfg.Repo = projutil.CheckAndGetProjectGoPkg()
		}
		s := &scaffold.Scaffold{}
		if err := writeConfig(s, cfg); err != nil {
			return err
		}
	}
	return nil
}
func writeConfig(s *scaffold.Scaffold, cfg *input.Config) error {
	log.Info("Writing new default CSV config.")
	csvCfg := &catalog.CSVConfig{
		Input: input.Input{Path: writeCSVConfigPath},
	}
	if err := s.Execute(cfg, csvCfg); err != nil {
		return fmt.Errorf("error scaffolding CSV config: %v", err)
	}
	return nil
}
