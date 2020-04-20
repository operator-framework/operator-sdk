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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetCRs parses a Bundle's CSV for CRs
func (c TestBundle) GetCRs() (crList []unstructured.Unstructured, err error) {

	// get CRs from CSV's alm-examples annotation, assume single bundle

	csv, err := c.Bundles[0].ClusterServiceVersion()
	if err != nil {
		return crList, fmt.Errorf("error in csv retrieval %s", err.Error())
	}

	if csv.GetAnnotations() == nil {
		return crList, nil
	}

	almExamples := csv.ObjectMeta.Annotations["alm-examples"]

	if almExamples == "" {
		return crList, nil
	}

	if len(c.Bundles) > 0 {
		var crInterfaces []map[string]interface{}
		err = json.Unmarshal([]byte(almExamples), &crInterfaces)
		if err != nil {
			return crList, err
		}
		for i := 0; i < len(crInterfaces); i++ {
			buff := new(bytes.Buffer)
			enc := json.NewEncoder(buff)
			err := enc.Encode(crInterfaces[i])
			if err != nil {
				return crList, err
			}
			obj := &unstructured.Unstructured{}
			if err := obj.UnmarshalJSON(buff.Bytes()); err != nil {
				return crList, err
			}
			crList = append(crList, *obj)
		}
	}

	return crList, err
}
