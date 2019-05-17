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

package writeconfig

import (
	"github.com/operator-framework/operator-sdk/pkg/config"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type writeConfigCmd struct {
	path string
	repo string
}

func NewCmd() *cobra.Command {
	c := writeConfigCmd{}

	wcCmd := &cobra.Command{
		Use:   "write-config",
		Short: "Writes a config file to disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			if c.path == "" {
				c.path = config.DefaultFileName
			}
			if err := config.WriteConfigAs(c.path); err != nil {
				return err
			}
			if c.repo == "" {
				log.Infof(`Config field "repo" not set. Ensure you set this field`+
					` before using %s for an Operator project.`, c.path)
			}
			return nil
		},
	}

	wcCmd.Flags().StringVar(&c.path, "path", "", "Path to write Operator SDK config file")
	wcCmd.Flags().StringVar(&c.repo, config.RepoOpt, "", "Project repository path, ex. github.com/operator-framework/operator-sdk. This flag is required")
	viper.BindPFlag(config.RepoOpt, wcCmd.Flags().Lookup(config.RepoOpt))

	return wcCmd
}
