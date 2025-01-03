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
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"

	scorecardannotations "github.com/operator-framework/operator-sdk/internal/annotations/scorecard"
	xunit "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/scorecard/xunit"
	"github.com/operator-framework/operator-sdk/internal/flags"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
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
	storageImage   string
	untarImage     string
	testOutput     string
	podSecurity    string
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
		PreRunE: func(_ *cobra.Command, args []string) (err error) {
			return c.validate(args)
		},
		RunE: func(_ *cobra.Command, args []string) (err error) {
			c.bundle = args[0]
			return c.run()
		},
	}

	scorecardCmd.Flags().StringVar(&c.kubeconfig, "kubeconfig", "", "kubeconfig path")
	scorecardCmd.Flags().StringVarP(&c.selector, "selector", "l", "", "label selector to determine which tests are run")
	scorecardCmd.Flags().StringVarP(&c.config, "config", "c", "", "path to scorecard config file")
	scorecardCmd.Flags().StringVarP(&c.namespace, "namespace", "n", "", "namespace to run the test images in")
	scorecardCmd.Flags().StringVarP(&c.outputFormat, "output", "o", "text",
		"Output format for results. Valid values: text, json, xunit")
	scorecardCmd.Flags().StringVar(&c.podSecurity, "pod-security", "legacy", "option to run scorecard with legacy pod security context")
	scorecardCmd.Flags().StringVarP(&c.serviceAccount, "service-account", "s", "default",
		"Service account to use for tests")
	scorecardCmd.Flags().BoolVarP(&c.list, "list", "L", false,
		"Option to enable listing which tests are run")
	scorecardCmd.Flags().BoolVarP(&c.skipCleanup, "skip-cleanup", "x", false,
		"Disable resource cleanup after tests are run")
	scorecardCmd.Flags().DurationVarP(&c.waitTime, "wait-time", "w", 30*time.Second,
		"seconds to wait for tests to complete. Example: 35s")
	// Please note that for Operator-sdk + Preflight + DCI integration in disconnected environments,
	// it is necessary to refer to storage-image and untar-image using their digests instead of tags.
	// If you need to make changes to these images, please ensure that you always use the digests.
	scorecardCmd.Flags().StringVarP(&c.storageImage, "storage-image", "b",
		"quay.io/operator-framework/scorecard-storage@sha256:a3bfda71281393c7794cabdd39c563fb050d3020fd0b642ea164646bdd39a0e2",
		"Storage image to be used by the Scorecard pod")
	// Use the digest of the latest scorecard-untar image
	scorecardCmd.Flags().StringVarP(&c.untarImage, "untar-image", "u",
		"quay.io/operator-framework/scorecard-untar@sha256:2e728c5e67a7f4dec0df157a322dd5671212e8ae60f69137463bd4fdfbff8747",
		"Untar image to be used by the Scorecard pod")
	scorecardCmd.Flags().StringVarP(&c.testOutput, "test-output", "t", "test-output",
		"Test output directory.")

	return scorecardCmd
}

func (c *scorecardCmd) printOutput(output v1alpha3.TestList) error {
	switch c.outputFormat {
	case "text":
		if len(output.Items) == 0 {
			fmt.Println("0 tests selected")
			return nil
		}
		for _, test := range output.Items {
			fmt.Println(test.MarshalText())
		}
	case "json":
		bytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json error: %v", err)
		}
		fmt.Printf("%s\n", string(bytes))
	case "xunit":
		xunitOutput := c.convertXunit(output)
		bytes, err := xml.MarshalIndent(xunitOutput, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal xml error: %v", err)
		}
		fmt.Printf("%s\n", string(bytes))
	default:
		return fmt.Errorf("invalid output format selected")
	}
	return nil
}

func (c *scorecardCmd) convertXunit(output v1alpha3.TestList) xunit.TestSuites {
	const (
		imagePropName        = "spec.image"
		entrypointPropName   = "spec.entrypoint"
		testPropName         = "labels.test"
		clusterPhasePropName = "labels.cluster-phase"
	)

	suites := make([]xunit.TestSuite, 0, len(output.Items))
	for i, item := range output.Items {
		suiteName, ok := item.Spec.Labels["test"]
		if !ok {
			suiteName = fmt.Sprintf("testsuite-%03d", i+1)
		}

		ts := xunit.NewSuite(suiteName)
		ts.AddProperty(imagePropName, item.Spec.Image)
		ts.AddProperty(entrypointPropName, strings.Join(item.Spec.Entrypoint, " "))
		ts.AddProperty(testPropName, suiteName)
		if phase, ok := item.Spec.Labels["cluster-phase"]; ok {
			ts.AddProperty(clusterPhasePropName, phase)
		}

		for _, tc := range item.Status.Results {
			switch tc.State {
			case v1alpha3.PassState:
				ts.AddSuccess(tc.Name, tc.CreationTimestamp.Time, tc.Log)
			case v1alpha3.FailState:
				ts.AddFailure(tc.Name, tc.CreationTimestamp.Time, tc.Log, strings.Join(tc.Errors, "\n"))
			case v1alpha3.ErrorState:
				ts.AddError(tc.Name, tc.CreationTimestamp.Time, tc.Log, strings.Join(tc.Errors, "\n"))
			}
		}

		suites = append(suites, ts)
	}

	return xunit.NewTestSuites("scorecard", suites)
}

func (c *scorecardCmd) run() (err error) {
	// Extract bundle image contents if bundle is inferred to be an image.
	if _, err = os.Stat(c.bundle); err != nil && errors.Is(err, os.ErrNotExist) {
		if c.bundle, err = extractBundleImage(c.bundle); err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := os.RemoveAll(c.bundle); err != nil {
				log.Error(err)
			}
		}()
	}

	metadata, _, err := registryutil.FindBundleMetadata(c.bundle)
	if err != nil {
		log.Fatal(err)
	}

	podSecFlag := true
	if c.podSecurity == "restricted" {
		podSecFlag = true
	} else if c.podSecurity == "legacy" {
		podSecFlag = false
	}

	o := scorecard.Scorecard{
		SkipCleanup: c.skipCleanup,
		PodSecurity: podSecFlag,
	}

	configPath := c.config
	if configPath == "" {
		configDir, hasDir := scorecardannotations.GetConfigDir(metadata)
		if !hasDir {
			configDir = filepath.FromSlash(scorecard.DefaultConfigDir)
		}
		configPath = filepath.Join(c.bundle, configDir, scorecard.ConfigFileName)
	}
	o.Config, err = scorecard.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("could not find config file %w", err)
	}

	o.Selector, err = labels.Parse(c.selector)
	if err != nil {
		return fmt.Errorf("could not parse selector %w", err)
	}

	var scorecardTests v1alpha3.TestList
	if c.list {
		scorecardTests = o.List()
	} else {
		runnerSA := c.serviceAccount
		if o.Config.ServiceAccount != "" {
			runnerSA = o.Config.ServiceAccount
		}
		runner := scorecard.PodTestRunner{
			ServiceAccount: runnerSA,
			Namespace:      scorecard.GetKubeNamespace(c.kubeconfig, c.namespace),
			BundlePath:     c.bundle,
			TestOutput:     c.testOutput,
			BundleMetadata: metadata,
			StorageImage:   c.storageImage,
			UntarImage:     c.untarImage,
		}

		// Only get the client if running tests.
		if runner.Client, runner.RESTConfig, err = scorecard.GetKubeClient(c.kubeconfig); err != nil {
			return fmt.Errorf("error getting kubernetes client: %w", err)
		}

		o.TestRunner = &runner

		ctx, cancel := context.WithTimeout(context.Background(), c.waitTime)
		defer cancel()

		scorecardTests, err = o.Run(ctx)
		if err != nil {
			// if we got a timeout; printout the test results if there are any
			if err == context.DeadlineExceeded {
				if errpo := c.printOutput(scorecardTests); errpo != nil {
					log.Fatal(errpo)
				}
			}
			return fmt.Errorf("error running tests %w", err)
		}
	}

	if err := c.printOutput(scorecardTests); err != nil {
		log.Fatal(err)
	}

	if hasFailingTest(scorecardTests) {
		os.Exit(1)
	}
	return nil
}

func hasFailingTest(list v1alpha3.TestList) bool {
	for _, t := range list.Items {
		for _, r := range t.Status.Results {
			if r.State != v1alpha3.PassState {
				return true
			}
		}
	}
	return false
}

func (c *scorecardCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a bundle image or directory argument is required")
	}
	return nil
}

// extractBundleImage returns bundleImage's path on disk post-extraction.
func extractBundleImage(bundleImage string) (string, error) {
	// Discard bundle extraction logs unless user sets verbose mode.
	logger := registryutil.DiscardLogger()
	if viper.GetBool(flags.VerboseOpt) {
		logger = log.WithFields(log.Fields{"bundle": bundleImage})
	}
	// FEAT: enable explicit local image extraction.
	return registryutil.ExtractBundleImage(context.TODO(), logger, bundleImage, false, false, false)
}
