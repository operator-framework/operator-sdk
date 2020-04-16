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

package genutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

type InternalError string

func (e InternalError) Error() string {
	return fmt.Sprintf("internal error: %s", string(e))
}

func GetCSVName(name, version string) string {
	return fmt.Sprintf("%s.v%s", name, version)
}

type File struct {
	*os.File
}

func Open(dir, fileName string) (*File, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, fileName), os.O_RDWR|os.O_CREATE, 0666)
	return &File{f}, err
}

func WriteObject(w io.Writer, obj interface{}) error {
	b, err := k8sutil.GetObjectBytes(obj, yaml.Marshal)
	if err != nil {
		return err
	}
	return write(w, b)
}

func WriteYAML(w io.Writer, obj interface{}) error {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	return write(w, b)
}

func write(w io.Writer, b []byte) error {
	if f, isFile := w.(*File); isFile {
		if err := f.Truncate(0); err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
	}
	_, err := w.Write(b)
	return err
}

// IsExist returns true if path exists on disk.
func IsExist(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil || errors.Is(err, os.ErrExist)
}

func IsNotExist(path string) bool {
	if path == "" {
		return true
	}
	_, err := os.Stat(path)
	return err != nil && errors.Is(err, os.ErrNotExist)
}
