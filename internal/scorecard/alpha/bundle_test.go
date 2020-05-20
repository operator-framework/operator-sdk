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

package alpha

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBundlePath(t *testing.T) {

	cases := []struct {
		bundlePathValue string
		expTarFile      string
		wantError       bool
	}{
		{"", "", true},
		{
			bundlePathValue: filepath.Join("testdata", "bundle"),
			expTarFile:      filepath.Join("testdata", "bundle.tar.gz"),
		},
	}

	for _, c := range cases {
		t.Run(c.bundlePathValue, func(t *testing.T) {
			r := PodTestRunner{}
			r.BundlePath = c.bundlePathValue
			bundleData, err := r.getBundleData()
			if err != nil && !c.wantError {
				t.Fatalf("Wanted result but got error: %v", err)
			} else if err == nil {
				if c.wantError {
					t.Fatalf("Wanted error but got no error")
				}

				expTarData, err := ioutil.ReadFile(c.expTarFile)
				if err != nil {
					t.Fatalf("Failed to read expected tar file: %v", err)
				}
				if !cmpTarFiles(t, expTarData, bundleData) {
					t.Error("Bundle tar file does not match the expected tar file")
				}
			}
		})
	}
}

func cmpTarFiles(t *testing.T, c1, c2 []byte) bool {
	r1, r2 := bytes.NewBuffer(c1), bytes.NewBuffer(c2)
	w1, w2 := &bytes.Buffer{}, &bytes.Buffer{}
	if err := untar(t, r1, w1); err != nil {
		t.Fatalf("Error untarring first content: %v", err)
	}
	if err := untar(t, r2, w2); err != nil {
		t.Fatalf("Error untarring second content: %v", err)
	}
	return reflect.DeepEqual(w1.String(), w2.String())
}

func untar(t *testing.T, r io.Reader, w io.Writer) (err error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		n, err := io.Copy(w, tr)
		if err != nil {
			return err
		}
		if n != hdr.Size {
			return fmt.Errorf("unexpected bytes written: wrote %d, want %d", n, hdr.Size)
		}
	}
	return nil

}
