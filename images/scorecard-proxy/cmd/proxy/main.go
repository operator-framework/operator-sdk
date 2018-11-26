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
	"flag"

	proxy "github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func main() {
	flag.Parse()
	logf.SetLogger(logf.ZapLogger(false))

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Namespace: namespace,
	})
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan error)

	// start the proxy
	err = proxy.Run(done, proxy.Options{
		Address:          "localhost",
		Port:             8888,
		KubeConfig:       mgr.GetConfig(),
		NoOwnerInjection: true,
		LogRequests:      true,
	})
	if err != nil {
		log.Fatalf("error starting proxy: %v", err)
	}

	// wait forever or exit on proxy crash/finish
	err = <-done
	if err == nil {
		log.Info("Exiting")
	} else {
		log.Fatal(err.Error())
	}
}
