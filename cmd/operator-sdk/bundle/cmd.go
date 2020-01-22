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

package bundle

import (
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Work with operator bundle metadata and bundle images",
		Long: `Generate operator bundle metadata and build operator bundle images, which
are used to manage operators in the Operator Lifecycle Manager.

More information on operator bundle images and metadata:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-bundle.md#docker`,
	}

	cmd.AddCommand(
		newCreateCmd(),
		newValidateCmd(),
	)
	return cmd
}

type bundleCmd struct {
	directory      string
	packageName    string
	imageTag       string
	imageBuilder   string
	defaultChannel string
	channels       []string
	generateOnly   bool
}

// cleanupFuncs returns a set of general funcs to clean up after a bundle
// subcommand.
func (c bundleCmd) cleanupFuncs() (fs []func()) {
	metaDir := filepath.Join(c.directory, bundle.MetadataDir)
	dockerFile := filepath.Join(c.directory, bundle.DockerFile)
	metaExists := isExist(metaDir)
	dockerFileExists := isExist(dockerFile)
	fs = append(fs,
		func() {
			if !metaExists {
				_ = os.RemoveAll(metaDir)
			}
		},
		func() {
			if !dockerFileExists {
				_ = os.RemoveAll(dockerFile)
			}
		})
	return fs
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
}
