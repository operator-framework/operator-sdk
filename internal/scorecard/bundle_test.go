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
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/registry"
)

var _ = Describe("Tarring a bundle", func() {
	Describe("getBundleData", func() {

		var (
			r          PodTestRunner
			err        error
			expTarPath = filepath.Join("testdata", "bundle.tar.gz")
			expTarball []byte
		)

		BeforeEach(func() {
			r = PodTestRunner{}
			expTarball, err = os.ReadFile(expTarPath)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with a valid on-disk bundle", func() {
			var (
				validBundlePath = filepath.Join("testdata", "bundle")
			)

			It("creates a tarball successfully", func() {
				r.BundlePath = validBundlePath
				r.BundleMetadata, _, err = registry.FindBundleMetadata(validBundlePath)
				Expect(err).ToNot(HaveOccurred())
				tarredBundleData, err := r.getBundleData()
				Expect(err).ToNot(HaveOccurred())
				cmpTarFilesHelper(expTarball, tarredBundleData)
			})
		})

		Context("with an invalid on-disk bundle", func() {
			It("returns an error", func() {
				_, err = r.getBundleData()
				Expect(err).To(HaveOccurred())
			})
		})
	})

})

// cmpTarFilesHelper compares the byte representation of two tarballs,
// by contents per matching header name and for non-intersecting header names.
func cmpTarFilesHelper(c1, c2 []byte) {
	r1, r2 := bytes.NewBuffer(c1), bytes.NewBuffer(c2)
	set1, err := untarToFileSet(r1)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	set2, err := untarToFileSet(r2)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for fileName, contents1 := range set1 {
		contents2, hasFileName := set2[fileName]
		ExpectWithOffset(1, hasFileName).To(BeTrue(), "second tarball does not have file %s", fileName)
		ExpectWithOffset(1, contents1.String()).To(Equal(contents2.String()),
			"contents of file %s differ in first and second tarballs", fileName)
		delete(set2, fileName)
	}
	ExpectWithOffset(1, set2).To(BeEmpty(), "second tarball has files not in the first")
}

// untarToFileSet reads a gizpped tarball from r and writes each object's bytes to a set, keyed by header name.
func untarToFileSet(r io.Reader) (map[string]*bytes.Buffer, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := gz.Close(); err != nil {
			fmt.Fprintln(GinkgoWriter, "warning: error closing tarball reader:", err)
		}
	}()
	tr := tar.NewReader(gz)
	fileSet := make(map[string]*bytes.Buffer)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		buf := &bytes.Buffer{}
		n, err := io.Copy(buf, tr)
		if err != nil {
			return nil, err
		}
		if n != hdr.Size {
			return nil, fmt.Errorf("unexpected bytes written: wrote %d, want %d", n, hdr.Size)
		}
		fileSet[path.Clean(hdr.Name)] = buf
	}

	return fileSet, nil
}
