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

package helm

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/helm/controller"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"github.com/operator-framework/operator-sdk/pkg/helm/watches"
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
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

// Run runs the helm operator
func Run(flags *hoflags.HelmOperatorFlags) error {
	printVersion()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config.")
		return err
	}

	// Set default manager options
	options := manager.Options{
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		NewClient: func(cache cache.Cache, config *rest.Config, options crclient.Options) (crclient.Client, error) {
			c, err := crclient.New(config, options)
			if err != nil {
				return nil, err
			}
			return &crclient.DelegatingClient{
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

	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "Failed to create a new manager.")
		return err
	}

	watches, err := watches.Load(flags.WatchesFile)
	if err != nil {
		log.Error(err, "Failed to create new manager factories.")
		return err
	}
	var gvks []schema.GroupVersionKind
	for _, w := range watches {
		// Register the controller with the factory.
		err := controller.Add(mgr, controller.WatchOptions{
			Namespace:               namespace,
			GVK:                     w.GroupVersionKind,
			ManagerFactory:          release.NewManagerFactory(mgr, w.ChartDir),
			ReconcilePeriod:         flags.ReconcilePeriod,
			WatchDependentResources: w.WatchDependentResources,
			OverrideValues:          w.OverrideValues,
			MaxWorkers:              flags.MaxWorkers,
		})
		if err != nil {
			log.Error(err, "Failed to add manager factory to controller.")
			return err
		}
		gvks = append(gvks, w.GroupVersionKind)
	}

	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		log.Error(err, "Failed to get operator name")
		return err
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, operatorName+"-lock")
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

	servicePorts := []v1.ServicePort{
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	_, err = metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info(err.Error())
	}

	// Start the Cmd
	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero.")
		os.Exit(1)
	}
	return nil
}
