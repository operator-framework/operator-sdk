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

package validate

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/bundle/validate/internal"
	"github.com/operator-framework/operator-sdk/internal/flags"
)

const (
	longHelp = `The 'operator-sdk bundle validate' command can validate both content and format of an operator bundle
image or an operator bundle directory on-disk containing operator metadata and manifests. This command will exit
with an exit code of 1 if any validation errors arise, and 0 if only warnings arise or all validators pass.

A valid bundle is defined by the bundle spec (linked below), therefore the default validator ensures a bundle conforms to
that spec. If you want to ensure that your bundle is valid for an optional superset of requirements such as to those
required to publish your operator on operatorhub.io, then you will need to run one or more supported optional validators.
Set '--list-optional' to list which optional validators are supported, and how they are grouped by label.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md

NOTE: if validating an image, the image must exist in a remote registry, not just locally.
`

	examples = `This example assumes you either have a *pullable* bundle image,
or something similar to the following operator bundle layout present locally:

  $ tree ./bundle
  ./bundle
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

To validate a local bundle:

  $ operator-sdk bundle validate ./bundle

To build and validate a *pullable* bundle image:

  $ operator-sdk bundle validate <some-registry>/<operator-bundle-name>:<tag>

To list and run optional validators, which are specified by a label selector:

  $ operator-sdk bundle validate --list-optional
  NAME           LABELS                     DESCRIPTION
  operatorhub    name=operatorhub           OperatorHub.io metadata validation.
                 suite=operatorframework

To validate a bundle against the entire suite of validators for Operator Framework, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional suite=operatorframework

The OperatorHub.io validator in the operatorframework optional suite allows you to validate that your manifests can work with a Kubernetes cluster of a particular version using the k8s-version optional key value:

  $ operator-sdk bundle validate ./bundle --select-optional suite=operatorframework --optional-values=k8s-version=1.22

To validate a bundle against the validator for operatorhub.io specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=operatorhub

This validator allows check the bundle against an specific Kubernetes cluster version using the k8s-version optional key value:

  $ operator-sdk bundle validate ./bundle --select-optional name=operatorhub --optional-values=k8s-version=1.22

[Deprecated] To validate a bundle against the (alpha) validator for Community Operators specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=community --optional-values=index-path=bundle.Dockerfile

To validate a bundle against the validator for Good Practices specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=good-practices

To validate a bundle against the (alpha) validator for Deprecated APIs specifically, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --select-optional name=alpha-deprecated-apis --optional-values=k8s-version=1.22

To validate a bundle against an external validator, in addition to required bundle validators:

  $ operator-sdk bundle validate ./bundle --alpha-select-external /path/to/external-validator[:/path/to/optional-second-validator]

To validate a bundle against the (alpha) validator for Multiple Architectures bundle validation, in addition to required bundle validators:

IMPORTANT: To use this option it is required to have access to pull the images defined on the CSV.

  $ operator-sdk bundle validate ./bundle --select-optional name=multiarch 

NOTE: The --optional-values can be used to inform the container-tools that should be used i.e. "--optional-values=container-tools=docker".
The valid values for the container-tools optional value are [docker, podman, none]. If no value is supplied then the command will default to using docker to inspect the images.
More info: https://github.com/operator-framework/api/blob/master/pkg/validation/internal/multiarch.go
`
)

// NewCmd returns a command that will validate an operator bundle.
func NewCmd() *cobra.Command {
	c := bundleValidateCmd{}
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validate an operator bundle",
		Long:    longHelp,
		Example: examples,
		RunE: func(_ *cobra.Command, args []string) (err error) {
			// Always print non-output logs to stderr as to not pollute actual command output.
			// Note that it allows the JSON result be redirected to the Stdout. E.g
			// if we run the command with `| jq . > result.json` the command will print just the logs
			// and the file will have only the JSON result.
			logger := createLogger(viper.GetBool(flags.VerboseOpt))

			if c.selector, err = labels.Parse(c.selectorRaw); err != nil {
				logger.Fatal(err)
			}

			if err = c.validate(args); err != nil {
				return fmt.Errorf("invalid command args: %v", err)
			}

			if c.listOptional {
				if err = c.list(); err != nil {
					logger.Fatal(err)
				}
				return nil
			}

			if c.selectorRaw == "name=community" {
				logger.Warnf("The Optional(stage: alpha) Community Operator bundle validator is deprecated and" +
					" will be removed in a future release. You can do these checks using the external validator: " +
					"https://github.com/redhat-openshift-ecosystem/ocp-olm-catalog-validator/")
			}

			result, err := c.run(logger, args[0])
			if err != nil {
				logger.Fatal(err)
			}
			failed, err := result.PrintWithFormat(c.outputFormat)
			if err != nil {
				logger.Fatal(err)
			}

			// if a test failed don't print that it was successful
			if failed {
				os.Exit(1)
			}

			logger.Info("All validation tests have completed successfully")
			return nil
		},
	}

	c.addToFlagSet(cmd.Flags())

	return cmd
}

// createLogger creates a new logrus Entry that is optionally verbose.
func createLogger(verbose bool) *log.Entry {
	logger := log.NewEntry(internal.NewLoggerTo(os.Stderr))
	if verbose {
		logger.Logger.SetLevel(log.DebugLevel)
	}
	return logger
}
