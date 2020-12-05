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

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/ansible"
	golang "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go"
	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/helm"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

func main() {
	// testdata is the path where all samples should be generate
	const testdata = "/testdata/"

	// binaryName allow inform the binary that should be used.
	// By default it is operator-sdk
	var binaryName string

	flag.StringVar(&binaryName, "bin", testutils.BinaryName, "Binary path that should be used")
	flag.Parse()

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	samplesPath := filepath.Join(currentPath, testdata)
	log.Infof("using the path: (%v)", samplesPath)

	log.Infof("creating Helm Memcached Sample")
	helm.GenerateMemcachedHelmSample(samplesPath)

	log.Infof("creating Ansible Memcached Sample")
	ansible.GenerateMemcachedAnsibleSample(samplesPath)

	log.Infof("creating Go Memcached Sample with Webhooks")
	golang.GenerateMemcachedGoWithWebhooksSample(samplesPath)
}
