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

package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/cmd/ansible-operator/run"
	"github.com/operator-framework/operator-sdk/internal/cmd/ansible-operator/version"
)

func main() {
<<<<<<< HEAD
	root := cobra.Command{
		Use: "ansible-operator",
=======
	f := &flags.Flags{}
	f.AddTo(pflag.CommandLine)
	pflag.Parse()
	logf.SetLogger(zap.Logger())

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
		if pflag.CommandLine.Lookup("leader-election-id").Changed {
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
>>>>>>> Added flags, added fields to runner struct
	}

	root.AddCommand(run.NewCmd())
	root.AddCommand(version.NewCmd())

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
