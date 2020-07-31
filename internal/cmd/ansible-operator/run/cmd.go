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
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapf "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/operator-framework/operator-sdk/internal/ansible/controller"
	"github.com/operator-framework/operator-sdk/internal/ansible/flags"
	"github.com/operator-framework/operator-sdk/internal/ansible/proxy"
	"github.com/operator-framework/operator-sdk/internal/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/internal/ansible/runner"
	"github.com/operator-framework/operator-sdk/internal/ansible/watches"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/internal/version"
)

var (
	metricsHost           = "0.0.0.0"
	log                   = logf.Log.WithName("cmd")
	healthProbePort int32 = 6789
)

func printVersion() {
	log.Info("Version",
		"Go Version", runtime.Version(),
		"GOOS", runtime.GOOS,
		"GOARCH", runtime.GOARCH,
		"ansible-operator", sdkVersion.Version)
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

	// Set default manager options
	// TODO: probably should expose the host & port as an environment variables
	options := manager.Options{
		HealthProbeBindAddress:  fmt.Sprintf("%s:%d", metricsHost, healthProbePort),
		MetricsBindAddress:      f.MetricsAddress,
		LeaderElection:          f.EnableLeaderElection,
		LeaderElectionID:        f.LeaderElectionID,
		LeaderElectionNamespace: f.LeaderElectionNamespace,
		NewClient: func(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
			c, err := client.New(config, options)
			if err != nil {
				return nil, err
			}
			return &client.DelegatingClient{
				Reader:       cache,
				Writer:       c,
				StatusClient: c,
			}, nil
		},
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

	err = setAnsibleEnvVars(f)
	if err != nil {
		log.Error(err, "Failed to set environment variable.")
		os.Exit(1)
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "Failed to create a new manager.")
		os.Exit(1)
	}

	cMap := controllermap.NewControllerMap()
	watches, err := watches.Load(f.WatchesFile, f.MaxConcurrentReconciles, f.AnsibleVerbosity)
	if err != nil {
		log.Error(err, "Failed to load watches.")
		os.Exit(1)
	}
	for _, w := range watches {
		runner, err := runner.New(w, f.AnsibleArgs)
		if err != nil {
			log.Error(err, "Failed to create runner")
			os.Exit(1)
		}

		ctr := controller.Add(mgr, controller.Options{
			GVK:                     w.GroupVersionKind,
			Runner:                  runner,
			ManageStatus:            w.ManageStatus,
			AnsibleDebugLogs:        getAnsibleDebugLog(),
			MaxConcurrentReconciles: w.MaxConcurrentReconciles,
			ReconcilePeriod:         w.ReconcilePeriod,
			Selector:                w.Selector,
		})
		if ctr == nil {
			log.Error(fmt.Errorf("failed to add controller for GVK %v", w.GroupVersionKind.String()), "")
			os.Exit(1)
		}

		cMap.Store(w.GroupVersionKind, &controllermap.Contents{Controller: *ctr,
			WatchDependentResources:     w.WatchDependentResources,
			WatchClusterScopedResources: w.WatchClusterScopedResources,
			OwnerWatchMap:               controllermap.NewWatchMap(),
			AnnotationWatchMap:          controllermap.NewWatchMap(),
		}, w.Blacklist)
	}

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	if err != nil {
		log.Error(err, "Failed to add Healthz check.")
	}

	done := make(chan error)

	// start the proxy
	err = proxy.Run(done, proxy.Options{
		Address:           "localhost",
		Port:              8888,
		KubeConfig:        mgr.GetConfig(),
		Cache:             mgr.GetCache(),
		RESTMapper:        mgr.GetRESTMapper(),
		ControllerMap:     cMap,
		OwnerInjection:    f.InjectOwnerRef,
		WatchedNamespaces: []string{namespace},
	})
	if err != nil {
		log.Error(err, "Error starting proxy.")
		os.Exit(1)
	}

	// start the operator
	go func() {
		done <- mgr.Start(signals.SetupSignalHandler())
	}()

	// wait for either to finish
	err = <-done
	if err != nil {
		log.Error(err, "Proxy or operator exited with error.")
		os.Exit(1)
	}
	log.Info("Exiting.")
}

// getAnsibleDebugLog return the value from the ANSIBLE_DEBUG_LOGS it order to
// print the full Ansible logs
func getAnsibleDebugLog() bool {
	const envVar = "ANSIBLE_DEBUG_LOGS"
	val := false
	if envVal, ok := os.LookupEnv(envVar); ok {
		if i, err := strconv.ParseBool(envVal); err != nil {
			log.Info("Could not parse environment variable as an boolean; using default value",
				"envVar", envVar, "default", val)
		} else {
			val = i
		}
	} else if !ok {
		log.Info("Environment variable not set; using default value", "envVar", envVar,
			envVar, val)
	}
	return val
}

// setAnsibleEnvVars will set environment variables based on CLI flags
func setAnsibleEnvVars(f *flags.Flags) error {
	if len(f.AnsibleRolesPath) > 0 {
		if err := os.Setenv(flags.AnsibleRolesPathEnvVar, f.AnsibleRolesPath); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %v", flags.AnsibleRolesPathEnvVar, err)
		}
		log.Info("Set the environment variable", "envVar", flags.AnsibleRolesPathEnvVar,
			"value", f.AnsibleRolesPath)
	}

	if len(f.AnsibleCollectionsPath) > 0 {
		if err := os.Setenv(flags.AnsibleCollectionsPathEnvVar, f.AnsibleCollectionsPath); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %v", flags.AnsibleCollectionsPathEnvVar, err)
		}
		log.Info("Set the environment variable", "envVar", flags.AnsibleCollectionsPathEnvVar,
			"value", f.AnsibleCollectionsPath)
	}
	return nil
}
