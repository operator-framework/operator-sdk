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

package scorecard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/operator-framework/operator-sdk/internal/flags"
	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
)

type scorecardCmd struct {
	bundle         string
	config         string
	kubeconfig     string
	namespace      string
	outputFormat   string
	selector       string
	serviceAccount string
	list           bool
	skipCleanup    bool
	waitTime       time.Duration
}

func NewCmd() *cobra.Command {
	c := scorecardCmd{}

	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Runs scorecard",
		// TODO: describe what the purpose of the command is, why someone would want
		// to run it, etc.
		Long: `Has flags to configure dsl, bundle, and selector. This command takes
one argument, either a bundle image or directory containing manifests and metadata.
If the argument holds an image tag, it must be present remotely.`,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.run()
		},
	}

	scorecardCmd.Flags().StringVar(&c.kubeconfig, "kubeconfig", "", "kubeconfig path")
	scorecardCmd.Flags().StringVarP(&c.selector, "selector", "l", "", "label selector to determine which tests are run")
	scorecardCmd.Flags().StringVarP(&c.config, "config", "c", "", "path to scorecard config file")
	scorecardCmd.Flags().StringVarP(&c.namespace, "namespace", "n", "default", "namespace to run the test images in")
	scorecardCmd.Flags().StringVarP(&c.outputFormat, "output", "o", "text",
		"Output format for results.  Valid values: text, json")
	scorecardCmd.Flags().StringVarP(&c.serviceAccount, "service-account", "s", "default",
		"Service account to use for tests")
	scorecardCmd.Flags().BoolVarP(&c.list, "list", "L", false,
		"Option to enable listing which tests are run")
	scorecardCmd.Flags().BoolVarP(&c.skipCleanup, "skip-cleanup", "x", false,
		"Disable resource cleanup after tests are run")
	scorecardCmd.Flags().DurationVarP(&c.waitTime, "wait-time", "w", time.Duration(30*time.Second),
		"seconds to wait for tests to complete. Example: 35s")

	return scorecardCmd
}

func (c *scorecardCmd) printOutput(output v1alpha3.Test) error {
	switch c.outputFormat {
	case "text":
		if len(output.Status.Results) == 0 {
			fmt.Println("0 tests selected")
			return nil
		}
		o, err := output.MarshalText()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("%s\n", o)
	case "json":
		bytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	default:
		return fmt.Errorf("invalid output format selected")
	}
	return nil

}

func (c *scorecardCmd) run() error {
	var err error
	// Extract bundle image contents if bundle is inferred to be an image.
	if _, err = os.Stat(c.bundle); err != nil && errors.Is(err, os.ErrNotExist) {
		// Discard bundle extraction logs unless user sets verbose mode.
		logger := log.NewEntry(discardLogger())
		if viper.GetBool(flags.VerboseOpt) {
			logger = log.WithFields(log.Fields{"bundle": c.bundle})
		}
		// FEAT: enable explicit local image extraction.
		if c.bundle, err = scorecard.ExtractBundleImage(context.TODO(), logger, c.bundle, false); err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := os.RemoveAll(c.bundle); err != nil {
				logger.Error(err)
			}
		}()
	}

	o := scorecard.Scorecard{
		SkipCleanup: c.skipCleanup,
	}

	configPath := c.config
	if configPath == "" {
		configPath = filepath.Join(c.bundle, "tests", "scorecard", "config.yaml")
	}
	o.Config, err = scorecard.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("could not find config file %w", err)
	}

	o.Selector, err = labels.Parse(c.selector)
	if err != nil {
		return fmt.Errorf("could not parse selector %w", err)
	}

	var scorecardTest v1alpha3.Test
	if c.list {
		scorecardTest, err = o.ListTests()
		if err != nil {
			return fmt.Errorf("error listing tests %w", err)
		}
	} else {
		runner := scorecard.PodTestRunner{
			ServiceAccount: c.serviceAccount,
			Namespace:      c.namespace,
			BundlePath:     c.bundle,
		}

		// Only get the client if running tests.
		if runner.Client, err = scorecard.GetKubeClient(c.kubeconfig); err != nil {
			return fmt.Errorf("error getting kubernetes client: %w", err)
		}

		o.TestRunner = &runner

		ctx, cancel := context.WithTimeout(context.Background(), c.waitTime)
		defer cancel()

		scorecardTest, err = o.RunTests(ctx)
		if err != nil {
			return fmt.Errorf("error running tests %w", err)
		}
	}

	return c.printOutput(scorecardTest)
}

func (c *scorecardCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a bundle image or directory argument is required")
	}
	c.bundle = args[0]

	return nil
}

// discardLogger returns a logger that throws away input.
func discardLogger() *log.Logger {
	logger := log.New()
	logger.SetOutput(ioutil.Discard)
	return logger
}
