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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapf "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/operator-framework/operator-sdk/internal/clientbuilder"
	"github.com/operator-framework/operator-sdk/internal/helm/controller"
	"github.com/operator-framework/operator-sdk/internal/helm/flags"
	"github.com/operator-framework/operator-sdk/internal/helm/metrics"
	"github.com/operator-framework/operator-sdk/internal/helm/release"
	"github.com/operator-framework/operator-sdk/internal/helm/watches"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/internal/version"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info("Version",
		"Go Version", runtime.Version(),
		"GOOS", runtime.GOOS,
		"GOARCH", runtime.GOARCH,
		"helm-operator", sdkVersion.Version,
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

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config.")
		os.Exit(1)
	}

	// Deprecated: OPERATOR_NAME environment variable is an artifact of the
	// legacy operator-sdk project scaffolding. Flag `--leader-election-id`
	// should be used instead.
	if operatorName, found := os.LookupEnv("OPERATOR_NAME"); found {
		log.Info("Environment variable OPERATOR_NAME has been deprecated, use --leader-election-id instead.")
		if cmd.Flags().Lookup("leader-election-id").Changed {
			log.Info("Ignoring OPERATOR_NAME environment variable since --leader-election-id is set")
		} else {
			f.LeaderElectionID = operatorName
		}
	}

	//todo: remove the following checks for 2.0.0 they are required just because of the flags deprecation
	if cmd.Flags().Changed("leader-elect") && cmd.Flags().Changed("enable-leader-election") {
		log.Error(errors.New("only one of --leader-elect and --enable-leader-election may be set"), "invalid flags usage")
		os.Exit(1)
	}

	if cmd.Flags().Changed("metrics-addr") && cmd.Flags().Changed("metrics-bind-address") {
		log.Error(errors.New("only one of --metrics-addr and --metrics-bind-address may be set"), "invalid flags usage")
		os.Exit(1)
	}

	// Set default manager options
	options := manager.Options{
		MetricsBindAddress:         f.MetricsBindAddress,
		HealthProbeBindAddress:     f.ProbeAddr,
		LeaderElection:             f.LeaderElection,
		LeaderElectionID:           f.LeaderElectionID,
		LeaderElectionResourceLock: resourcelock.ConfigMapsResourceLock,
		LeaderElectionNamespace:    f.LeaderElectionNamespace,
		ClientBuilder:              clientbuilder.NewUnstructedCached(),
	}

	namespace, found := os.LookupEnv(k8sutil.WatchNamespaceEnvVar)
	log = log.WithValues("Namespace", namespace)
	if found {
		if namespace == metav1.NamespaceAll {
			log.Info("Watching all namespaces.")
			options.Namespace = metav1.NamespaceAll
		} else {
			if strings.Contains(namespace, ",") {
				log.Info("Watching multiple namespaces.")
				options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
			} else {
				log.Info("Watching single namespace.")
				options.Namespace = namespace
			}
		}
	} else {
		log.Info(fmt.Sprintf("%v environment variable not set. Watching all namespaces.",
			k8sutil.WatchNamespaceEnvVar))
		options.Namespace = metav1.NamespaceAll
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

	ws, err := watches.Load(f.WatchesFile)
	if err != nil {
		log.Error(err, "Failed to create new manager factories.")
		os.Exit(1)
	}
	for _, w := range ws {
		// Register the controller with the factory.
		err := controller.Add(mgr, controller.WatchOptions{
			Namespace:               namespace,
			GVK:                     w.GroupVersionKind,
			ManagerFactory:          release.NewManagerFactory(mgr, w.ChartDir),
			ReconcilePeriod:         f.ReconcilePeriod,
			WatchDependentResources: *w.WatchDependentResources,
			OverrideValues:          w.OverrideValues,
			MaxConcurrentReconciles: f.MaxConcurrentReconciles,
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
