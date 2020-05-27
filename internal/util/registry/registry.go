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

package registry

import (
	"fmt"
	"io/ioutil"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"gopkg.in/yaml.v2"
)

const (
	defaultPermission = 0644
)

// RewriteAnnotationsYaml unmarshalls the specified yaml file, appends the content and
// converts it again to yaml.
func RewriteAnnotationsYaml(filename, directory string, content map[string]string) error {

	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	annotationsYaml := &bundle.AnnotationMetadata{}
	if err := yaml.Unmarshal(f, annotationsYaml); err != nil {
		return fmt.Errorf("error parsing annotations file: %v", err)
	}

	// Append the contents to annotationsYaml
	for key, val := range content {
		annotationsYaml.Annotations[key] = val
	}

	file, err := yaml.Marshal(annotationsYaml)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, []byte(file), defaultPermission)
	if err != nil {
		return fmt.Errorf("error writing modified contents to annotations file, %v", err)
	}

	return nil

}
