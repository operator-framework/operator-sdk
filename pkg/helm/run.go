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
	"fmt"
	"os"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/controller"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
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

	namespace, found := os.LookupEnv(k8sutil.WatchNamespaceEnvVar)
	if found {
		log.Info("Watching single namespace", "namespace", namespace)
	} else {
		log.Info(fmt.Sprintf("%v environment variable not set. This operator is watching all namespaces.",
			k8sutil.WatchNamespaceEnvVar))
		namespace = metav1.NamespaceAll
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		return err
	}

	// Create Tiller's storage backend and kubernetes client
	storageBackend := storage.Init(driver.NewMemory())
	tillerKubeClient, err := client.NewFromManager(mgr)
	if err != nil {
		return err
	}

	factories, err := release.NewManagerFactoriesFromFile(storageBackend, tillerKubeClient, flags.WatchesFile)
	if err != nil {
		return err
	}

	for gvk, factory := range factories {
		// Register the controller with the factory.
		err := controller.Add(mgr, controller.WatchOptions{
			Namespace:               namespace,
			GVK:                     gvk,
			ManagerFactory:          factory,
			ReconcilePeriod:         flags.ReconcilePeriod,
			WatchDependentResources: true,
		})
		if err != nil {
			return err
		}
	}

	// Start the Cmd
	return mgr.Start(signals.SetupSignalHandler())
}
