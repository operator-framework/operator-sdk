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

package ansible

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	proxy "github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	metricsHost               = "0.0.0.0"
	log                       = logf.Log.WithName("cmd")
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

// Run will start the ansible operator and proxy, blocking until one of them
// returns.
func Run(flags *aoflags.AnsibleOperatorFlags) error {
	printVersion()

	namespace, found := os.LookupEnv(k8sutil.WatchNamespaceEnvVar)
	log = log.WithValues("Namespace", namespace)
	if found {
		log.Info("Watching namespace.")
	} else {
		log.Info(fmt.Sprintf("%v environment variable not set. This operator is watching all namespaces.",
			k8sutil.WatchNamespaceEnvVar))
		namespace = metav1.NamespaceAll
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config.")
		return err
	}
	// TODO: probably should expose the host & port as an environment variables
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
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
	})
	if err != nil {
		log.Error(err, "Failed to create a new manager.")
		return err
	}

	var gvks []schema.GroupVersionKind
	cMap := controllermap.NewControllerMap()
	watches, err := watches.Load(flags.WatchesFile, flags.MaxWorkers, flags.AnsibleVerbosity)
	if err != nil {
		log.Error(err, "Failed to load watches.")
		return err
	}
	for _, w := range watches {
		runner, err := runner.New(w)
		if err != nil {
			log.Error(err, "Failed to create runner")
			return err
		}

		ctr := controller.Add(mgr, controller.Options{
			GVK:             w.GroupVersionKind,
			Runner:          runner,
			ManageStatus:    w.ManageStatus,
			MaxWorkers:      w.MaxWorkers,
			ReconcilePeriod: w.ReconcilePeriod,
		})
		if ctr == nil {
			return fmt.Errorf("failed to add controller for GVK %v", w.GroupVersionKind.String())
		}

		cMap.Store(w.GroupVersionKind, &controllermap.Contents{Controller: *ctr,
			WatchDependentResources:     w.WatchDependentResources,
			WatchClusterScopedResources: w.WatchClusterScopedResources,
			OwnerWatchMap:               controllermap.NewWatchMap(),
			AnnotationWatchMap:          controllermap.NewWatchMap(),
		})
		gvks = append(gvks, w.GroupVersionKind)
	}

	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		log.Error(err, "Failed to get the operator name")
		return err
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), operatorName+"-lock")
	if err != nil {
		log.Error(err, "Failed to become leader.")
		return err
	}

	// Generates operator specific metrics based on the GVKs.
	// It serves those metrics on "http://metricsHost:operatorMetricsPort".
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, []string{namespace}, gvks, metricsHost, operatorMetricsPort)
	if err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	// TODO: probably should expose the port as an environment variable
	_, err = metrics.CreateMetricsService(context.TODO(), cfg, servicePorts)
	if err != nil {
		log.Error(err, "Exposing metrics port failed.")
		return err
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
		OwnerInjection:    flags.InjectOwnerRef,
		WatchedNamespaces: []string{namespace},
	})
	if err != nil {
		log.Error(err, "Error starting proxy.")
		return err
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
	return nil
}
