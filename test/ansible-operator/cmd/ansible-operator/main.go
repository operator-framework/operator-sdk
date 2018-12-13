// Copyright 2018 The Operator-SDK Authors
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
	"os"
	"runtime"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/operator"
	proxy "github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/sirupsen/logrus"
)

var defaultReconcilePeriod = pflag.String("reconcile-period", "1m", "default reconcile period for controllers")

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	pflag.Parse()
	logf.SetLogger(logf.ZapLogger(false))

	d, err := time.ParseDuration(*defaultReconcilePeriod)
	if err != nil {
		logrus.Fatalf("failed to parse reconcile-period: %v", err)
	}

	namespace, found := os.LookupEnv(k8sutil.WatchNamespaceEnvVar)
	if found {
		logrus.Infof("Watching %v namespace.", namespace)
	} else {
		logrus.Infof("%v environment variable not set. This operator is watching all namespaces.",
			k8sutil.WatchNamespaceEnvVar)
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Namespace: namespace,
	})
	if err != nil {
		log.Fatal(err)
	}

	printVersion()
	done := make(chan error)

	// start the proxy
	err = proxy.Run(done, proxy.Options{
		Address:    "localhost",
		Port:       8888,
		KubeConfig: mgr.GetConfig(),
		RESTMapper: mgr.GetRESTMapper(),
		Cache:      mgr.GetCache(),
	})
	if err != nil {
		logrus.Fatalf("error starting proxy: %v", err)
	}

	// start the operator
	go operator.Run(done, mgr, "/opt/ansible/watches.yaml", d)

	// wait for either to finish
	err = <-done
	if err == nil {
		logrus.Info("Exiting")
	} else {
		logrus.Fatal(err.Error())
	}
}
