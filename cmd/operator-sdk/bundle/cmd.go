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
	"log"
	"path/filepath"

	"github.com/spf13/cobra"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

//nolint:structcheck
type bundleCmd struct {
	directory      string
	packageName    string
	imageTag       string
	imageBuilder   string
	defaultChannel string
	channels       string
	generateOnly   bool
}

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

func getDefaults() *bundleCmd {
	c := &bundleCmd{
		imageBuilder:   "docker",
		defaultChannel: "stable",
		channels:       "stable",
		generateOnly:   false,
	}

	if kbutil.IsConfigExist() {
		cfg, err := kbutil.ReadConfig()
		if err != nil {
			log.Fatal(err)
		}
		c.packageName = filepath.Base(cfg.Repo)
		c.directory = filepath.Join("config", "olm-catalog", c.packageName, "manifests")
	} else {
		c.packageName = filepath.Base(projutil.MustGetwd())
		// For generating CLI docs.
		if c.packageName == "operator-sdk" {
			c.packageName = "test-operator"
		}
		c.directory = filepath.Join("deploy", "olm-catalog", c.packageName, "manifests")
	}

	return c
}
