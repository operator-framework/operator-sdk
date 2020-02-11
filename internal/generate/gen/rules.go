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

package gen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
)

// OutputToCachedDirectory configures a generator runtime to output files
// to a directory. The output option rule string is formatted as follows:
//
// - output:<generator>:<form>:dir (per-generator output)
// - output:<form>:dir (default output)
//
// where <generator> is the generator's registered string name and <form>
// is the output rule's registered form string. See the CRD generator for
// an example of how this is used.
type OutputToCachedDirectory struct {
	Dir string
}

var _ genall.OutputRule = OutputToCachedDirectory{}

// Open is used to generate a CRD manifest in cache at path.
func (o OutputToCachedDirectory) Open(_ *loader.Package, path string) (io.WriteCloser, error) {
	if cache == nil {
		return nil, fmt.Errorf("error opening %s in output rule: cache must be set", path)
	}
	if err := cache.MkdirAll(o.Dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error mkdir %v in output rule: %v", o.Dir, err)
	}
	dirPath := filepath.Join(o.Dir, path)
	wc, err := cache.Create(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error creating %v in output rule: %v", dirPath, err)
	}
	return wc, nil
}
