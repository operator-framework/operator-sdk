// Copyright 2020 The Operator-SDK Authors
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
	"os"
	"path/filepath"

	testutils "github.com/operator-framework/operator-sdk/test/utils"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/helm"
	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		binaryName string
	)

	flag.StringVar(&binaryName, "bin", testutils.BinaryName, "Binary path that should be used")
	flag.Parse()

	current, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	samplesPath := filepath.Join(current, "/testdata/helm/memcached-operator")
	log.Infof("using the path: (%v)", samplesPath)

	log.Infof("starting to generate helm memcached sample")
	ctx, err := pkg.NewSampleContext(binaryName, samplesPath, "GO111MODULE=on")
	pkg.CheckError("error to generate helm memcached sample", err)

	log.Infof("creating Memcached Sample")
	helm.GenerateMemcachedHelmSample(&ctx)
}
