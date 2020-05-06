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
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

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
	"sigs.k8s.io/controller-runtime/pkg/healthz"

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
	healthProbePort     int32 = 6789
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

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config.")
		return err
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
			GVK:              w.GroupVersionKind,
			Runner:           runner,
			ManageStatus:     w.ManageStatus,
			AnsibleDebugLogs: getAnsibleDebugLog(),
			MaxWorkers:       w.MaxWorkers,
			ReconcilePeriod:  w.ReconcilePeriod,
		})
		if ctr == nil {
			return fmt.Errorf("failed to add controller for GVK %v", w.GroupVersionKind.String())
		}

		cMap.Store(w.GroupVersionKind, &controllermap.Contents{Controller: *ctr,
			WatchDependentResources:     w.WatchDependentResources,
			WatchClusterScopedResources: w.WatchClusterScopedResources,
			OwnerWatchMap:               controllermap.NewWatchMap(),
			AnnotationWatchMap:          controllermap.NewWatchMap(),
		}, w.Blacklist)
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

	addMetrics(context.TODO(), cfg, gvks)
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

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config, gvks []schema.GroupVersionKind) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCRMetrics(cfg, operatorNs, gvks); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
		return
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics takes GVKs retrieved from watches and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, operatorNs string, gvks []schema.GroupVersionKind) error {
	// The metrics will be generated from the namespaces which are returned here.
	// NOTE that passing nil or an empty list of namespaces in GenerateAndServeCRMetrics will result in an error.
	ns, err := kubemetrics.GetNamespacesForMetrics(operatorNs)
	if err != nil {
		return err
	}

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, gvks, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
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
