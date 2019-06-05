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

package genutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"

	log "github.com/sirupsen/logrus"
)

// ParseGroupVersions parses the layout of pkg/apis to return a map of
// API groups to versions.
func parseGroupVersions() (map[string][]string, error) {
	gvs := make(map[string][]string)
	groups, err := ioutil.ReadDir(scaffold.ApisDir)
	if err != nil {
		return nil, fmt.Errorf("could not read pkg/apis directory to find api Versions: %v", err)
	}

	for _, g := range groups {
		if g.IsDir() {
			groupDir := filepath.Join(scaffold.ApisDir, g.Name())
			versions, err := ioutil.ReadDir(groupDir)
			if err != nil {
				return nil, fmt.Errorf("could not read %s directory to find api Versions: %v", groupDir, err)
			}

			gvs[g.Name()] = make([]string, 0)
			for _, v := range versions {
				if v.IsDir() {
					// Ignore directories that do not contain any files, so generators
					// do not get empty directories as arguments.
					verDir := filepath.Join(groupDir, v.Name())
					files, err := ioutil.ReadDir(verDir)
					if err != nil {
						return nil, fmt.Errorf("could not read %s directory to find api Versions: %v", verDir, err)
					}
					for _, f := range files {
						if !f.IsDir() && filepath.Ext(f.Name()) == ".go" {
							gvs[g.Name()] = append(gvs[g.Name()], filepath.ToSlash(v.Name()))
							break
						}
					}
				}
			}
		}
	}

	if len(gvs) == 0 {
		return nil, fmt.Errorf("no groups or versions found in %s", scaffold.ApisDir)
	}
	return gvs, nil
}

// createFQAPIs return a slice of all fully qualified pkg + groups + versions
// of pkg and gvs in the format "pkg/groupA/v1".
func createFQAPIs(pkg string, gvs map[string][]string) (apis []string) {
	for g, vs := range gvs {
		for _, v := range vs {
			apis = append(apis, path.Join(pkg, g, v))
		}
	}
	return apis
}

// generateWithHeaderFile runs f with a header file path as an arguemnt.
// If there is no project boilerplate.go.txt file, an empty header file is
// created and its path passed as the argument.
// generateWithHeaderFile is meant to be used with Kubernetes code generators.
func generateWithHeaderFile(f func(string) error) (err error) {
	i, err := (&scaffold.Boilerplate{}).GetInput()
	if err != nil {
		return err
	}
	hf := i.Path
	if _, err := os.Stat(hf); os.IsNotExist(err) {
		if hf, err = createEmptyTmpFile(); err != nil {
			return err
		}
		defer func() {
			if err = os.RemoveAll(hf); err != nil {
				log.Error(err)
			}
		}()
	}
	return f(hf)
}

func createEmptyTmpFile() (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}
