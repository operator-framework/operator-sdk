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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

func ValidateVersion(version string) error {
	v, err := semver.Parse(version)
	if err != nil {
		return fmt.Errorf("%s is not a valid semantic version: %v", version, err)
	}
	// Ensures numerical values composing csvVersion don't contain leading 0's,
	// ex. 01.01.01
	if v.String() != version {
		return fmt.Errorf("provided CSV version %s contains bad values (parses to %s)", version, v)
	}
	return nil
}

func IsPipeReader() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe != 0
}

func PluginKeyToOperatorType(pluginKey string) projutil.OperatorType {
	switch {
	case strings.HasPrefix(pluginKey, "go"):
		return projutil.OperatorTypeGo
	}
	return ""
}

func WriteCRDs(w io.Writer, crds ...v1beta1.CustomResourceDefinition) error {
	for _, crd := range crds {
		if err := writeCRD(w, crd); err != nil {
			return err
		}
	}
	return nil
}

func WriteCRDFiles(dir string, crds ...v1beta1.CustomResourceDefinition) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, crd := range crds {
		if err := writeCRDFile(dir, crd); err != nil {
			return err
		}
	}
	return nil
}

func writeCRDFile(dir string, crd v1beta1.CustomResourceDefinition) error {
	file := fmt.Sprintf("%s_%s.yaml", crd.Spec.Group, crd.Spec.Names.Plural)
	f, err := os.Create(filepath.Join(dir, file))
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCRD(f, crd)
}

func writeCRD(w io.Writer, crd v1beta1.CustomResourceDefinition) error {
	b, err := yaml.Marshal(crd)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

type multiManifestWriter struct {
	io.Writer
}

func (w *multiManifestWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(append([]byte("\n---\n"), bytes.TrimSpace(b)...))
}

func NewMultiManifestWriter(w io.Writer) io.Writer {
	return &multiManifestWriter{w}
}
