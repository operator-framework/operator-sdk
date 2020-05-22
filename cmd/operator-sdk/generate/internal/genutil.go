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
	"errors"
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

// ValidateVersion returns an error if version is not a strict semantic version.
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

// IsPipeReader returns true if stdin is an open pipe, i.e. the caller can
// accept input from stdin.
func IsPipeReader() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe != 0
}

// PluginKeyToOperatorType converts a plugin key string to an operator project
// type.
// TODO(estroz): this can probably be made more robust by checking known
// plugin keys directly.
func PluginKeyToOperatorType(pluginKey string) projutil.OperatorType {
	switch {
	case strings.HasPrefix(pluginKey, "go"):
		return projutil.OperatorTypeGo
	}
	return ""
}

// WriteCRDs writes each CustomResourceDefinition in crds to w.
func WriteCRDs(w io.Writer, crds ...v1beta1.CustomResourceDefinition) error {
	for _, crd := range crds {
		if err := writeCRD(w, crd); err != nil {
			return err
		}
	}
	return nil
}

// WriteCRDFiles creates dir then writes each CustomResourceDefinition in crds
// to a file in dir.
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

// writeCRDFile marshals crd to bytes and writes them to dir in a file named
// <full group>_<resource>.yaml.
func writeCRDFile(dir string, crd v1beta1.CustomResourceDefinition) error {
	file := fmt.Sprintf("%s_%s.yaml", crd.Spec.Group, crd.Spec.Names.Plural)
	f, err := os.Create(filepath.Join(dir, file))
	if err != nil {
		return err
	}
	defer f.Close()
	return writeCRD(f, crd)
}

// writeCRD marshals crd to bytes and writes them to w.
func writeCRD(w io.Writer, crd v1beta1.CustomResourceDefinition) error {
	b, err := yaml.Marshal(crd)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// multiManifestWriter writes a multi-part manifest by prepending "---"
// to the argument of io.Writer.Write().
type multiManifestWriter struct {
	io.Writer
}

func (w *multiManifestWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(append([]byte("\n---\n"), bytes.TrimSpace(b)...))
}

// NewMultiManifestWriter returns a multi-part manifest writer. Use this writer
// if writing a package or bundle to stdout or a single file.
func NewMultiManifestWriter(w io.Writer) io.Writer {
	return &multiManifestWriter{w}
}

// IsNotExist returns true if path does not exist on disk.
func IsNotExist(path string) bool {
	if path == "" {
		return true
	}
	_, err := os.Stat(path)
	return err != nil && errors.Is(err, os.ErrNotExist)
}
