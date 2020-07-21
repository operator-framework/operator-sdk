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
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	"github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
)

var (
	metricsHost           = "0.0.0.0"
	log                   = logf.Log.WithName("cmd")
	metricsPort     int32 = 8383
	healthProbePort int32 = 6789
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
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

	// Set default manager options
	// TODO: probably should expose the host & port as an environment variables
	options := manager.Options{
		HealthProbeBindAddress: fmt.Sprintf("%s:%d", metricsHost, healthProbePort),
		MetricsBindAddress:     fmt.Sprintf("%s:%d", metricsHost, metricsPort),
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
		runner, err := runner.New(w)
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

	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		log.Error(err, "Failed to get the operator name")
		os.Exit(1)
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), operatorName+"-lock")
	if err != nil {
		log.Error(err, "Failed to become leader.")
		os.Exit(1)
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
