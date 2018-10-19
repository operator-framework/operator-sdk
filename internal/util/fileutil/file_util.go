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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/util/util.go

package fileutil

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

const (
	// file modes
	DefaultDirFileMode  = 0750
	DefaultFileMode     = 0644
	DefaultExecFileMode = 0755

	DefaultFileFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
)

// FileWriter is a io wrapper to write files
type FileWriter struct {
	Fs afero.Fs

	once sync.Once
}

// WriteCloser returns a WriteCloser to write to given path
func (fw *FileWriter) WriteCloser(path string, mode os.FileMode) (io.Writer, error) {
	fw.once.Do(func() {
		fw.Fs = afero.NewOsFs()
	})

	dir := filepath.Dir(path)
	err := fw.Fs.MkdirAll(dir, DefaultDirFileMode)
	if err != nil {
		return nil, err
	}

	fi, err := fw.Fs.OpenFile(path, DefaultFileFlags, mode)
	if err != nil {
		return nil, err
	}

	return fi, nil
}

// WriteFile write given content to the file path
func (fw *FileWriter) WriteFile(filePath string, content []byte) error {
	fw.once.Do(func() {
		fw.Fs = afero.NewOsFs()
	})

	f, err := fw.WriteCloser(filePath, DefaultFileMode)
	if err != nil {
		return fmt.Errorf("failed to create %s: %v", filePath, err)
	}

	if c, ok := f.(io.Closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	_, err = f.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", filePath, err)
	}

	return nil
}
