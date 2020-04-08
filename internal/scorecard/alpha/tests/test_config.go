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

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation/errors"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestConfig holds the bundle contents to be tested
type TestConfig struct {
	PackageManifest registry.PackageManifest
	BundleErrors    []errors.ManifestResult
	Bundles         []*registry.Bundle
	CRs             []unstructured.Unstructured
}

// GetConfig parses a Bundle from a given on-disk path returning a TestConfig
func GetConfig(bundlePath string) (cfg TestConfig, err error) {

	cfg.PackageManifest, cfg.Bundles, cfg.BundleErrors = manifests.GetManifestsDir(bundlePath)

	// get CRs from CSV's alm-examples annotation, assume single bundle
	cfg.CRs = make([]unstructured.Unstructured, 0)

	csv, err := cfg.Bundles[0].ClusterServiceVersion()
	if err != nil {
		return cfg, fmt.Errorf("error in csv retrieval %s", err.Error())
	}
	almExamples := csv.ObjectMeta.Annotations["alm-examples"]

	if almExamples == "" {
		log.Printf("no alm-examples were found, so no CRs")
		return cfg, nil
	}

	if len(cfg.Bundles) > 0 {
		var crInterfaces []map[string]interface{}
		err = json.Unmarshal([]byte(almExamples), &crInterfaces)
		if err != nil {
			log.Printf("error unmarshalling CRs from alm-examples %s\n", err.Error())
		}
		for i := 0; i < len(crInterfaces); i++ {
			buff := new(bytes.Buffer)
			enc := json.NewEncoder(buff)
			err := enc.Encode(crInterfaces[i])
			if err != nil {
				log.Printf("error encoding CRs from alm-examples %s\n", err.Error())
			} else {
				obj := &unstructured.Unstructured{}
				if err := obj.UnmarshalJSON(buff.Bytes()); err != nil {
				} else {
					cfg.CRs = append(cfg.CRs, *obj)
				}
			}
		}
	}

	return cfg, err
}
