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

package scorecard

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func WriteToTar(tw *tar.Writer, r io.Reader, hdr *tar.Header) error {
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := io.Copy(tw, r)
	return err
}

// WritePathsToTar walks paths to create tar file tarName
func WritePathsToTar(tw *tar.Writer, paths []string) (err error) {
	// walk each specified path and add encountered file to tar
	for _, path := range paths {
		path = filepath.Clean(path)

		walker := func(file string, finfo os.FileInfo, err error) error {
			if err != nil || file == path {
				return err
			}

			// fill in header info using func FileInfoHeader
			hdr, err := tar.FileInfoHeader(finfo, finfo.Name())
			if err != nil {
				return err
			}

			relFilePath := file
			if filepath.IsAbs(path) {
				relFilePath, err = filepath.Rel(path, file)
				if err != nil {
					return err
				}
			}
			// ensure header has relative file path
			hdr.Name = strings.TrimPrefix(relFilePath, path+string(filepath.Separator))
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			// if path is a dir, dont continue
			if finfo.IsDir() {
				return nil
			}

			// add file to tar
			srcFile, err := os.Open(file)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, srcFile)
			if err := srcFile.Close(); err != nil {
				log.Error(err)
			}
			return err
		}

		// build tar
		err = filepath.Walk(path, walker)
		if err != nil {
			return fmt.Errorf("failed to add %s to tar: %w", path, err)
		}
	}
	return nil
}

// untar a file into a location
func UntarFile(tarName, target string) (err error) {
	tarFile, err := os.Open(tarName)
	if err != nil {
		return err
	}
	defer func() {
		if err := tarFile.Close(); err != nil {
			log.Error(err)
		}
	}()

	absPath, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	var tr *tar.Reader
	if isFileGzipped(tarName) {
		gz, err := gzip.NewReader(tarFile)
		if err != nil {
			return err
		}
		defer func() {
			if err := gz.Close(); err != nil {
				log.Error(err)
			}
		}()
		tr = tar.NewReader(gz)
	} else {
		tr = tar.NewReader(tarFile)
	}

	// untar each segment
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// determine proper file path info
		finfo := hdr.FileInfo()
		fileName := hdr.Name
		if filepath.IsAbs(fileName) {
			fileName, err = filepath.Rel("/", fileName)
			if err != nil {
				return err
			}
		}
		absFileName := filepath.Join(absPath, fileName)

		if finfo.Mode().IsDir() {
			if err := os.MkdirAll(absFileName, 0755); err != nil {
				return err
			}
			continue
		}

		// create new file with original file mode
		file, err := os.OpenFile(absFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, finfo.Mode().Perm())
		if err != nil {
			return err
		}
		n, cpErr := io.Copy(file, tr)
		if closeErr := file.Close(); closeErr != nil { // close file immediately
			return err
		}
		if cpErr != nil {
			return cpErr
		}
		if n != finfo.Size() {
			return fmt.Errorf("unexpected bytes written: wrote %d, want %d", n, finfo.Size())
		}
	}
	return nil

}

// isFileGzipped returns true if file is compressed with gzip.
func isFileGzipped(file string) bool {
	return strings.HasSuffix(file, ".gz") || strings.HasSuffix(file, ".gzip")
}
