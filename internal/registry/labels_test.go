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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = Describe("Labels", func() {
	Describe("FindBundleMetadata", func() {
		var (
			fs          afero.Fs
			err         error
			defaultPath = "/bundle/metadata/annotations.yaml"
		)

		Context("with valid annotations contents", func() {
			var (
				metadata      LabelsMap
				path, expPath string
			)
			BeforeEach(func() {
				fs = afero.NewMemMapFs()
			})

			// Location
			It("finds registry metadata in the default location", func() {
				expPath = defaultPath
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})
			It("finds registry metadata in the a custom file name", func() {
				expPath = "/bundle/metadata/my-metadata.yaml"
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})
			It("finds registry metadata in a custom single-depth location", func() {
				expPath = "/bundle/my-dir/my-metadata.yaml"
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})
			It("finds registry metadata in a custom multi-depth location", func() {
				expPath = "/bundle/my-parent-dir/my-dir/annotations.yaml"
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})
			It("returns registry metadata from default path when metadata is also in another location", func() {
				expPath = defaultPath
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				writeMetadataHelper(fs, "/bundle/other-metadata/annotations.yaml", annotationsStringValidNoRegLabels)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})
			It("returns registry metadata from the first path, when metadata is also in another location", func() {
				expPath = "/bundle/custom1/annotations.yaml"
				writeMetadataHelper(fs, expPath, annotationsStringValidV1)
				writeMetadataHelper(fs, "/bundle/custom2/annotations.yaml", annotationsStringValidNoRegLabels)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidV1))
			})

			// Format
			It("finds non-registry metadata", func() {
				expPath = defaultPath
				writeMetadataHelper(fs, defaultPath, annotationsStringValidNoRegLabels)
				metadata, path, err = findBundleMetadata(fs, "/bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expPath))
				Expect(metadata).To(BeEquivalentTo(annotationsValidNoRegLabels))
			})
		})

		Context("with invalid annotations contents", func() {
			BeforeEach(func() {
				fs = afero.NewMemMapFs()
			})

			It("returns a YAML error", func() {
				writeMetadataHelper(fs, defaultPath, annotationsStringInvalidBadIndent)
				_, _, err = findBundleMetadata(fs, "/bundle")
				// err should contain both of the following parts.
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("error unmarshalling potential bundle metadata %s: ", defaultPath)))
				Expect(err.Error()).To(ContainSubstring("yaml: line 2: found character that cannot start any token"))
			})
			It("returns an error for no metadata file (empty file)", func() {
				writeMetadataHelper(fs, defaultPath, annotationsStringInvalidEmpty)
				_, _, err = findBundleMetadata(fs, "/bundle")
				Expect(err).To(MatchError("metadata not found in /bundle"))
			})
			It("returns an error for no metadata file (invalid top-level key)", func() {
				writeMetadataHelper(fs, defaultPath, annotationsStringInvalidTopKey)
				_, _, err = findBundleMetadata(fs, "/bundle")
				Expect(err).To(MatchError("metadata not found in /bundle"))
			})
			It("returns an error for no labels in a metadata file", func() {
				writeMetadataHelper(fs, defaultPath, annotationsStringInvalidNoLabels)
				_, _, err = findBundleMetadata(fs, "/bundle")
				Expect(err).To(MatchError("metadata not found in /bundle"))
			})
		})
	})

})

func writeMetadataHelper(fs afero.Fs, path, contents string) {
	ExpectWithOffset(1, fs.MkdirAll(filepath.Dir(path), 0755)).Should(Succeed())
	ExpectWithOffset(1, afero.WriteFile(fs, path, []byte(contents), 0666)).Should(Succeed())
}

var annotationsValidV1 = LabelsMap{
	"operators.operatorframework.io.bundle.mediatype.v1": "registry+v1",
	"operators.operatorframework.io.bundle.metadata.v1":  "metadata/",
	"foo": "bar",
}

const annotationsStringValidV1 = `annotations:
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  foo: bar
`

var annotationsValidNoRegLabels = LabelsMap{
	"foo": "bar",
	"baz": "buf",
}

const annotationsStringValidNoRegLabels = `annotations:
  foo: bar
  baz: buf
`

const annotationsStringInvalidBadIndent = `annotations:
	operators.operatorframework.io.bundle.mediatype.v1: registry+v1
`

const annotationsStringInvalidEmpty = ``

const annotationsStringInvalidNoLabels = `annotations:
`

const annotationsStringInvalidTopKey = `not-annotations:
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  foo: bar
`
