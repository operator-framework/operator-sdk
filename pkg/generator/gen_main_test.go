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

package generator

import (
	"bytes"
	"testing"
)

const mainExp = `package main

import (
	"context"
	"runtime"
	"net/http"

	stub "github.com/example-inc/app-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Prometheus metrics port
const promPort = ":9090"

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
	logrus.Infof("operator prometheus port :%s", promPort)
}

func main() {
	printVersion()

	http.Handle("/metrics", promhttp.Handler())
	logrus.Fatalf("%s", http.ListenAndServe(promPort, nil))

	resource := "app.example.com/v1alpha1"
	kind := "AppService"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}
	resyncPeriod := 5
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
`

func TestGenMain(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderMainFile(buf, appRepoPath, appAPIVersion, appKind); err != nil {
		t.Error(err)
		return
	}

	if mainExp != buf.String() {
		t.Errorf(errorMessage, mainExp, buf.String())
	}
}
