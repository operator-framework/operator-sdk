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
	"os"
	"runtime"

	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/ansible/operator"
	proxy "github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func printVersion() {
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Infof("Version of operator-sdk: %v", sdkVersion.Version)
}

// Run will start the ansible operator and proxy, blocking until one of them
// returns.
func Run(flags *aoflags.AnsibleOperatorFlags) {
	logf.SetLogger(logf.ZapLogger(false))

	namespace, found := os.LookupEnv(k8sutil.WatchNamespaceEnvVar)
	if found {
		log.Infof("Watching %v namespace.", namespace)
	} else {
		log.Infof("%v environment variable not set. This operator is watching all namespaces.",
			k8sutil.WatchNamespaceEnvVar)
		namespace = metav1.NamespaceAll
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Namespace: namespace,
	})
	if err != nil {
		log.Fatal(err)
	}

	printVersion()
	done := make(chan error)
	cMap := proxy.NewControllerMap()

	// start the proxy
	err = proxy.Run(done, proxy.Options{
		Address:           "localhost",
		Port:              8888,
		KubeConfig:        mgr.GetConfig(),
		Cache:             mgr.GetCache(),
		RESTMapper:        mgr.GetRESTMapper(),
		ControllerMap:     cMap,
		WatchedNamespaces: []string{namespace},
	})
	if err != nil {
		log.Fatalf("Error starting proxy: (%v)", err)
	}

	// start the operator
	go operator.Run(done, mgr, flags, cMap)

	// wait for either to finish
	err = <-done
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Exiting")
}
