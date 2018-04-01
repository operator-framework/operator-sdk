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

// mainTmpl is the template for cmd/main.go.
const mainTmpl = `package main

import (
	"context"
	"runtime"

	stub "{{.StubImport}}"
	sdk "{{.OperatorSDKImport}}"
	sdkVersion "{{.SDKVersionImport}}"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()
	sdk.Watch("{{.APIVersion}}", "{{.Kind}}", "default", 5)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
`
