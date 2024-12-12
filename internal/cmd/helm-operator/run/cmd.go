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

package run

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapf "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	helmClient "github.com/operator-framework/operator-sdk/internal/helm/client"
	"github.com/operator-framework/operator-sdk/internal/helm/controller"
	"github.com/operator-framework/operator-sdk/internal/helm/flags"
	"github.com/operator-framework/operator-sdk/internal/helm/metrics"
	"github.com/operator-framework/operator-sdk/internal/helm/release"
	"github.com/operator-framework/operator-sdk/internal/helm/watches"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/internal/version"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	version := sdkVersion.GitVersion
	if version == "unknown" {
		version = sdkVersion.Version
	}
	log.Info("Version",
		"Go Version", runtime.Version(),
		"GOOS", runtime.GOOS,
		"GOARCH", runtime.GOARCH,
		"helm-operator", version,
		"commit", sdkVersion.GitCommit)
}

func NewCmd() *cobra.Command {
	f := &flags.Flags{}
	zapfs := flag.NewFlagSet("zap", flag.ExitOnError)
	opts := &zapf.Options{}
	opts.BindFlags(zapfs)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the operator",
		Args: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flag("metrics-require-rbac").Value.String() == "true" && cmd.Flag("metrics-secure").Value.String() == "false" {
				return errors.New("--metrics-secure flag is required when --metrics-require-rbac is present")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, _ []string) {
			logf.SetLogger(zapf.New(zapf.UseFlagOptions(opts)))
			run(cmd, f)
		},
	}

	f.AddTo(cmd.Flags())
	cmd.Flags().AddGoFlagSet(zapfs)
	return cmd
}

func run(cmd *cobra.Command, f *flags.Flags) {
	printVersion()
	metrics.RegisterBuildInfo(crmetrics.Registry)

	// Load config options from the config at f.ManagerConfigPath.
	// These options will not override those set by flags.
	var (
		options manager.Options
		err     error
	)

	exitIfUnsupported(options)

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config.")
		os.Exit(1)
	}

	// TODO(2.0.0): remove
	// Deprecated: OPERATOR_NAME environment variable is an artifact of the
	// legacy operator-sdk project scaffolding. Flag `--leader-election-id`
	// should be used instead.
	if operatorName, found := os.LookupEnv("OPERATOR_NAME"); found {
		log.Info("Environment variable OPERATOR_NAME has been deprecated, use --leader-election-id instead.")
		if cmd.Flags().Changed("leader-election-id") {
			log.Info("Ignoring OPERATOR_NAME environment variable since --leader-election-id is set")
		} else if options.LeaderElectionID == "" {
			// Only set leader election ID using OPERATOR_NAME if unset everywhere else,
			// since this env var is deprecated.
			options.LeaderElectionID = operatorName
		}
	}

	//TODO(2.0.0): remove the following checks. they are required just because of the flags deprecation
	if cmd.Flags().Changed("leader-elect") && cmd.Flags().Changed("enable-leader-election") {
		log.Error(errors.New("only one of --leader-elect and --enable-leader-election may be set"), "invalid flags usage")
		os.Exit(1)
	}

	if cmd.Flags().Changed("metrics-addr") && cmd.Flags().Changed("metrics-bind-address") {
		log.Error(errors.New("only one of --metrics-addr and --metrics-bind-address may be set"), "invalid flags usage")
		os.Exit(1)
	}

	// Set default manager options
	options = f.ToManagerOptions(options)

	if options.Scheme == nil {
		options.Scheme = apimachruntime.NewScheme()
	}

	ws, err := watches.Load(f.WatchesFile)
	if err != nil {
		log.Error(err, "Failed to load watches file.")
		os.Exit(1)
	}

	configureWatchNamespaces(&options, log)
	err = configureSelectors(&options, ws, options.Scheme)
	if err != nil {
		log.Error(err, "Failed to configure default selectors for caching")
		os.Exit(1)
	}
	if options.NewClient == nil {
		options.NewClient = client.New
	}

	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "Failed to create a new manager.")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	acg, err := helmClient.NewActionConfigGetter(mgr.GetConfig(), mgr.GetRESTMapper(), mgr.GetLogger())
	if err != nil {
		log.Error(err, "Failed to create Helm action config getter")
		os.Exit(1)
	}
	for _, w := range ws {
		// Register the controller with the factory.
		reconcilePeriod := f.ReconcilePeriod
		if w.ReconcilePeriod.Duration != time.Duration(0) {
			reconcilePeriod = w.ReconcilePeriod.Duration
		}

		err := controller.Add(mgr, controller.WatchOptions{
			GVK:                     w.GroupVersionKind,
			ManagerFactory:          release.NewManagerFactory(mgr, acg, w.ChartDir),
			ReconcilePeriod:         reconcilePeriod,
			WatchDependentResources: *w.WatchDependentResources,
			OverrideValues:          w.OverrideValues,
			SuppressOverrideValues:  f.SuppressOverrideValues,
			MaxConcurrentReconciles: f.MaxConcurrentReconciles,
			Selector:                w.Selector,
			DryRunOption:            w.DryRunOption,
		})
		if err != nil {
			log.Error(err, "Failed to add manager factory to controller.")
			os.Exit(1)
		}
	}

	// Start the Cmd
	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero.")
		os.Exit(1)
	}
}

// exitIfUnsupported prints an error containing unsupported field names and exits
// if any of those fields are not their default values.
func exitIfUnsupported(options manager.Options) {
	// The below options are webhook-specific, which is not supported by helm.
	if options.WebhookServer != nil {
		log.Error(errors.New("webhook configurations set in manager options"), "unsupported configuration")
		os.Exit(1)
	}
}

func configureWatchNamespaces(options *manager.Options, log logr.Logger) {
	namespaces := splitNamespaces(os.Getenv(k8sutil.WatchNamespaceEnvVar))

	namespaceConfigs := make(map[string]cache.Config)
	if len(namespaces) != 0 {
		log.Info("Watching namespaces", "namespaces", namespaces)
		for _, namespace := range namespaces {
			namespaceConfigs[namespace] = cache.Config{}
			if namespace == metav1.NamespaceAll {
				namespaceConfigs[namespace] = cache.Config{
					LabelSelector: labels.Everything(),
				}
			}
		}
	} else {
		log.Info("Watching all namespaces")
		// in order to properly establish cluster level watches
		// we need to override the default label selectors configured
		// in later config steps
		namespaceConfigs[metav1.NamespaceAll] = cache.Config{
			LabelSelector: labels.Everything(),
		}
	}

	options.Cache.DefaultNamespaces = namespaceConfigs
}

func splitNamespaces(namespaces string) []string {
	list := strings.Split(namespaces, ",")
	var out []string
	for _, ns := range list {
		trimmed := strings.TrimSpace(ns)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func configureSelectors(opts *manager.Options, ws []watches.Watch, sch *apimachruntime.Scheme) error {
	selectorsByObject := map[client.Object]cache.ByObject{}
	chartNames := make([]string, 0, len(ws))
	for _, w := range ws {
		sch.AddKnownTypeWithName(w.GroupVersionKind, &unstructured.Unstructured{})

		crObj := &unstructured.Unstructured{}
		crObj.SetGroupVersionKind(w.GroupVersionKind)
		sel, err := metav1.LabelSelectorAsSelector(&w.Selector)
		if err != nil {
			return fmt.Errorf("unable to parse watch selector for %s: %v", w.GroupVersionKind, err)
		}
		selectorsByObject[crObj] = cache.ByObject{Label: sel}

		chrt, err := loader.LoadDir(w.ChartDir)
		if err != nil {
			return fmt.Errorf("unable to load chart for %s: %v", w.GroupVersionKind, err)
		}
		chartNames = append(chartNames, chrt.Name())

	}
	req, err := labels.NewRequirement("helm.sdk.operatorframework.io/chart", selection.In, chartNames)
	if err != nil {
		return fmt.Errorf("unable to create label requirement for cache default selector: %v", err)
	}
	defaultSelector := labels.NewSelector().Add(*req)

	opts.Cache.ByObject = selectorsByObject
	opts.Cache.DefaultLabelSelector = defaultSelector
	return nil
}
