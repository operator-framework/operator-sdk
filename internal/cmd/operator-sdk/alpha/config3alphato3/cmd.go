// Copyright 2021 The Operator-SDK Authors
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

package config3alphato3

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config-3alpha-to-3",
		Short: "Convert your PROJECT config file from version 3-alpha to 3",
		Long: `Your PROJECT file contains config data specified by some version.
This version is not a kubernetes-style version. In general, alpha and beta config versions
are unstable and support for them is dropped once a stable version is released.
The 3-alpha version has recently become stable (3), and therefore is no longer
supported by operator-sdk v1.5+. This command is intended to migrate 3-alpha PROJECT files
to 3 with as few manual modifications required as possible.
`,
		RunE: func(_ *cobra.Command, _ []string) (err error) {
			cfgBytes, err := os.ReadFile("PROJECT")
			if err != nil {
				return fmt.Errorf("%v (config-3alpha-to-3 must be run from project root)", err)
			}

			if ver, err := getConfigVersion(cfgBytes); err == nil && ver != v3alpha {
				fmt.Println("Your PROJECT config file is not convertible at version", ver)
				return nil
			}

			b, err := convertConfig3AlphaTo3(cfgBytes)
			if err != nil {
				return err
			}
			if err := os.WriteFile("PROJECT", b, 0666); err != nil {
				return err
			}

			fmt.Println("Your PROJECT config file has been converted from version 3-alpha to 3. " +
				"Please make sure all config data is correct.")

			return nil
		},
	}
}

// RootPersistentPreRun prints a helpful message on any exit caused by kubebuilder's
// config unmarshal step finding "3-alpha", since the CLI will not recognize this version.
// Add this to the root command (`operator-sdk`).
var RootPersistentPreRun = func(_ *cobra.Command, _ []string) {
	if cfgBytes, err := os.ReadFile("PROJECT"); err == nil {
		if ver, err := getConfigVersion(cfgBytes); err == nil && ver == v3alpha {
			log.Warn("Config version 3-alpha has been stabilized as 3, and 3-alpha is no longer supported. " +
				"Run `operator-sdk alpha config-3alpha-to-3` to upgrade your PROJECT config file to version 3",
			)
		}
	}
}

func getConfigVersion(b []byte) (string, error) {
	var verObj struct {
		Version string `json:"version"`
	}
	return verObj.Version, yaml.Unmarshal(b, &verObj)
}
