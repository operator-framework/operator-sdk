// Copyright 2026 The Operator-SDK Authors
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

package bundle

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var _ = Describe("formatCSVs", func() {
	var (
		dir     string
		csvObj  map[string]interface{}
		csvPath string
	)

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		csvPath = filepath.Join(dir, "memcached-operator.clusterserviceversion.yaml")
		csvObj = map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "ClusterServiceVersion",
			"metadata": map[string]interface{}{
				"name": "memcached-operator.v0.0.1",
				// Strings a YAML 1.1 parser would read as another type if they
				// were left unquoted, plus a timestamp like the one in createdAt.
				"annotations": map[string]interface{}{
					"bool-like":  "yes",
					"on-like":    "on",
					"octal-like": "0755",
					"float-like": "1e3",
					"null-like":  "null",
					"empty":      "",
					"createdAt":  "2026-09-23T00:00:00Z",
				},
			},
			"spec": map[string]interface{}{
				"description": "First line.\n\n## Heading\n\n- item: with colon\n",
				"installModes": []interface{}{
					map[string]interface{}{"supported": true, "type": "OwnNamespace"},
				},
				"relatedImages": []interface{}{
					map[string]interface{}{"image": "busybox@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662", "name": "manager"},
				},
			},
		}

		// Write the CSV the way pinImages does, through operator-manifest-tools.
		csv, err := pullspec.NewOperatorCSV(csvPath, &unstructured.Unstructured{Object: csvObj}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(csvPath, nil, 0o644)).To(Succeed())
		Expect(csv.Dump(nil)).To(Succeed())
	})

	It("re-encodes a CSV the same way an unpinned bundle writes it", func() {
		want, err := yaml.Marshal(csvObj)
		Expect(err).NotTo(HaveOccurred())

		Expect(formatCSVs(dir)).To(Succeed())

		got, err := os.ReadFile(csvPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(got)).To(Equal(string(want)))
	})

	It("follows symlinked manifests that point outside the directory", func() {
		shared := GinkgoT().TempDir()
		target := filepath.Join(shared, "configmap.yaml")
		Expect(os.WriteFile(target, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm\n"), 0o644)).To(Succeed())
		Expect(os.Symlink(target, filepath.Join(dir, "configmap.yaml"))).To(Succeed())

		Expect(formatCSVs(dir)).To(Succeed())
	})

	It("leaves manifests that are not CSVs untouched", func() {
		other := filepath.Join(dir, "service.yaml")
		content := "apiVersion: v1\nkind: Service\nmetadata:\n  name: svc\nspec:\n  ports:\n    - port: 80\n"
		Expect(os.WriteFile(other, []byte(content), 0o644)).To(Succeed())

		Expect(formatCSVs(dir)).To(Succeed())

		got, err := os.ReadFile(other)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(got)).To(Equal(content))
	})
})
