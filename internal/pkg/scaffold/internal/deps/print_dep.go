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

package deps

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/toml"
)

func PrintDepGopkgTOML(tmpl string) error {
	gopkgData := make(map[string]interface{})
	_, err := toml.Decode(tmpl, &gopkgData)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 16, 8, 0, '\t', 0)
	_, err = w.Write([]byte("NAME\tVERSION\tBRANCH\tREVISION\t\n"))
	if err != nil {
		return err
	}

	constraintList, ok := gopkgData["constraint"]
	if !ok {
		return errors.New("constraints not found")
	}
	for _, dep := range constraintList.([]map[string]interface{}) {
		err = writeDepRow(w, dep)
		if err != nil {
			return err
		}
	}
	overrideList, ok := gopkgData["override"]
	if !ok {
		return errors.New("overrides not found")
	}
	for _, dep := range overrideList.([]map[string]interface{}) {
		err = writeDepRow(w, dep)
		if err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	requiredList, ok := gopkgData["required"]
	if !ok {
		return errors.New("required list not found")
	}
	pl, err := json.MarshalIndent(requiredList, "", " ")
	if err != nil {
		return err
	}
	_, err = buf.Write([]byte(fmt.Sprintf("\nrequired = %v", string(pl))))
	if err != nil {
		return err
	}

	fmt.Println(buf.String())

	return nil
}

func writeDepRow(w *tabwriter.Writer, dep map[string]interface{}) error {
	name := dep["name"].(string)
	ver, col := "", 0
	if v, ok := dep["version"]; ok {
		ver, col = v.(string), 1
	} else if v, ok = dep["branch"]; ok {
		ver, col = v.(string), 2
	} else if v, ok = dep["revision"]; ok {
		ver, col = v.(string), 3
	} else {
		return fmt.Errorf("no version, revision, or branch found for %s", name)
	}

	_, err := w.Write([]byte(name + strings.Repeat("\t", col) + ver + "\t\n"))
	return err
}
