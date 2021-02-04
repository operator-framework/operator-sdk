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

	"github.com/blang/semver/v4"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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
		return fmt.Errorf("version %s contains bad values (parses to %s)", version, v)
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

// WriteObjects writes each object in objs to w.
func WriteObjects(w io.Writer, objs ...client.Object) error {
	for _, obj := range objs {
		if err := writeObject(w, obj); err != nil {
			return err
		}
	}
	return nil
}

// WriteObjectsToFiles creates dir then writes each object in objs to a file in dir.
func WriteObjectsToFiles(dir string, objs ...client.Object) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	seenFiles := make(map[string]struct{})
	for _, obj := range objs {
		var fileName string
		switch t := obj.(type) {
		case *apiextv1.CustomResourceDefinition:
			fileName = makeCRDFileName(t.Spec.Group, t.Spec.Names.Plural)
		case *apiextv1beta1.CustomResourceDefinition:
			fileName = makeCRDFileName(t.Spec.Group, t.Spec.Names.Plural)
		default:
			fileName = makeObjectFileName(t)
		}

		if _, hasFile := seenFiles[fileName]; hasFile {
			return fmt.Errorf("duplicate file cannot be written: %s", fileName)
		}
		if err := writeObjectToFile(dir, obj, fileName); err != nil {
			return err
		}
		seenFiles[fileName] = struct{}{}
	}
	return nil
}

func makeCRDFileName(group, resource string) string {
	return fmt.Sprintf("%s_%s.yaml", group, resource)
}

func makeObjectFileName(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Group == "" {
		return fmt.Sprintf("%s_%s_%s.yaml", obj.GetName(), gvk.Version, strings.ToLower(gvk.Kind))
	}
	return fmt.Sprintf("%s_%s_%s_%s.yaml", obj.GetName(), gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
}

// writeObjectToFile marshals crd to bytes and writes them to dir in file.
func writeObjectToFile(dir string, obj interface{}, fileName string) error {
	f, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return err
	}
	defer f.Close()
	return writeObject(f, obj)
}

// writeObject marshals crd to bytes and writes them to w.
func writeObject(w io.Writer, obj interface{}) error {
	b, err := yaml.Marshal(obj)
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

// IsExist returns true if path exists on disk.
func IsExist(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil || errors.Is(err, os.ErrExist)
}
