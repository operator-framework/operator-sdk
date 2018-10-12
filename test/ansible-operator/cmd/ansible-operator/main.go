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
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	proxy "github.com/operator-framework/operator-sdk/pkg/ansible/proxy"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	flag.Parse()
	logf.SetLogger(logf.ZapLogger(false))

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Fatal(err)
	}

	printVersion()
	done := make(chan error)

	// start the proxy
	proxy.RunProxy(done, proxy.Options{
		Address:    "localhost",
		Port:       8888,
		KubeConfig: mgr.GetConfig(),
	})

	// start the operator
	go runSDK(done, mgr)

	// wait for either to finish
	err = <-done
	if err == nil {
		logrus.Info("Exiting")
	} else {
		logrus.Fatal(err.Error())
	}
}

func runSDK(done chan error, mgr manager.Manager) {
	watches, err := runner.NewFromWatches("/opt/ansible/watches.yaml")
	if err != nil {
		logrus.Error("Failed to get watches")
		done <- err
		return
	}
	rand.Seed(time.Now().Unix())
	c := signals.SetupSignalHandler()

	for gvk, runner := range watches {
		controller.Add(mgr, controller.Options{
			GVK:    gvk,
			Runner: runner,
		})
	}
	log.Fatal(mgr.Start(c))
	done <- nil
}
