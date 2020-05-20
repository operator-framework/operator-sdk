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

package execentrypoint

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
)

// newRunAnsibleCmd returns a command that will run an ansible operator.
func newRunAnsibleCmd() *cobra.Command {
	var flags *aoflags.AnsibleOperatorFlags
	runAnsibleCmd := &cobra.Command{
		Use:   "ansible",
		Short: "Runs as an ansible operator",
		Long: `Runs as an ansible operator. This is intended to be used when running
in a Pod inside a cluster. Developers wanting to run their operator locally
should use 'run local' instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logf.SetLogger(zap.Logger())
			if err := setAnsibleEnvVars(flags); err != nil {
				log.Fatal(err)
			}
			if err := ansible.Run(flags); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
	flags = aoflags.AddTo(runAnsibleCmd.Flags())

	return runAnsibleCmd
}

// setAnsibleEnvVars will set Ansible-defined environment variables from CLI flags.
func setAnsibleEnvVars(flags *aoflags.AnsibleOperatorFlags) error {
	if flags != nil {
		if len(flags.AnsibleRolesPath) > 0 {
			if err := os.Setenv(aoflags.AnsibleRolesPathEnvVar, flags.AnsibleRolesPath); err != nil {
				return fmt.Errorf("failed to set %s environment variable: (%v)", aoflags.AnsibleRolesPathEnvVar, err)
			}
			log.Infof("Set the value %v for environment variable %v.",
				flags.AnsibleRolesPath, aoflags.AnsibleRolesPathEnvVar)
		}
		if len(flags.AnsibleCollectionsPath) > 0 {
			if err := os.Setenv(aoflags.AnsibleCollectionsPathEnvVar, flags.AnsibleCollectionsPath); err != nil {
				return fmt.Errorf("failed to set %s environment variable: (%v)", aoflags.AnsibleCollectionsPathEnvVar, err)
			}
			log.Infof("Set the value %v for environment variable %v.",
				flags.AnsibleCollectionsPath, aoflags.AnsibleCollectionsPathEnvVar)
		}
	}
	return nil
}
