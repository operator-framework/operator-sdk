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

package catalog

import (
	"path/filepath"

	"github.com/coreos/go-semver/semver"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
	catalog "github.com/operator-framework/operator-sdk/pkg/scaffold/olm-catalog"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var csvVersion string

func NewGenCSVCmd() *cobra.Command {
	csvCmd := &cobra.Command{
		Use:   "gen-csv",
		Short: "Generates a Cluster Service Version yaml file for the operator",
		Long: `The operator-sdk olm-catalog gen-csv command generates a Cluster Service
Version (CSV) yaml manifest file for the operator. This file is used to publish
the operator to the OLM Catalog. A CSV semantic version is supplied via the
--csv-version flag.

Configure CSV generation by writing a config file 'deploy/olm-catalog/csv-config.yaml`,
		Run: csvFunc,
	}

	csvCmd.Flags().StringVar(&csvVersion, "csv-version", "", "Semantic version of the CSV")
	csvCmd.MarkFlagRequired("csv-version")

	return csvCmd
}

func csvFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		log.Fatal("gen-csv command doesn't accept any arguments")
	}

	verifyOLMCatalogFlags()

	absProjectPath := projutil.MustGetwd()
	cfg := &input.Config{
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	if projutil.GetOperatorType() == projutil.OperatorTypeGo {
		cfg.Repo = projutil.CheckAndGetProjectGoPkg()
	}

	log.Infof("Generating CSV manifest version %s", csvVersion)

	s := &scaffold.Scaffold{}
	err := s.Execute(cfg,
		&catalog.CSV{CSVVersion: csvVersion},
	)
	if err != nil {
		log.Fatalf("build catalog scaffold failed: (%v)", err)
	}
}

func verifyOLMCatalogFlags() {
	v, err := semver.NewVersion(csvVersion)
	if err != nil {
		log.Fatalf("%s is not a valid semantic version: (%v)", csvVersion, err)
	}
	if v.String() != csvVersion {
		log.Fatalf("provided CSV version %s contains bad values (parses to %s)", csvVersion, v)
	}
}
