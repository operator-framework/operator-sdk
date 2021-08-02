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

package k8sutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// GetDisplayName turns a project dir name in any of {snake, chain, camel}
// cases, hierarchical dot structure, or space-delimited into a
// space-delimited, title'd display name.
// Ex. "another-_AppOperator_againTwiceThrice More"
// ->  "Another App Operator Again Twice Thrice More"
func GetDisplayName(name string) string {
	for _, sep := range ".-_ " {
		splitName := strings.Split(name, string(sep))
		for i := 0; i < len(splitName); i++ {
			if splitName[i] == "" {
				splitName = append(splitName[:i], splitName[i+1:]...)
				i--
			} else {
				splitName[i] = strings.TrimSpace(splitName[i])
			}
		}
		name = strings.Join(splitName, " ")
	}
	splitName := strings.Split(name, " ")
	for i, word := range splitName {
		temp := word
		o := 0
		for j, r := range word {
			if unicode.IsUpper(r) {
				if j > 0 && !unicode.IsUpper(rune(word[j-1])) {
					index := j + o
					temp = temp[0:index] + " " + temp[index:]
					o++
				}
			}
		}
		splitName[i] = temp
	}
	return strings.TrimSpace(strings.Title(strings.Join(splitName, " ")))
}

// GetTypeMetaFromBytes gets the type and object metadata from b. b is assumed
// to be a single Kubernetes resource manifest.
func GetTypeMetaFromBytes(b []byte) (t metav1.TypeMeta, err error) {
	u := unstructured.Unstructured{}
	r := bytes.NewReader(b)
	dec := yaml.NewYAMLOrJSONDecoder(r, 8)
	// There is only one YAML doc if there are no more bytes to be read or EOF
	// is hit.
	if err := dec.Decode(&u); err == nil && r.Len() != 0 {
		return t, errors.New("error getting TypeMeta from bytes: more than one manifest in file")
	} else if err != nil && err != io.EOF {
		return t, fmt.Errorf("error getting TypeMeta from bytes: %v", err)
	}
	return metav1.TypeMeta{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
	}, nil
}

// dns1123LabelRegexp defines the character set allowed in a DNS 1123 label.
var dns1123LabelRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")

// FormatOperatorNameDNS1123 ensures name is DNS1123 label-compliant by
// replacing all non-compliant UTF-8 characters with "-".
func FormatOperatorNameDNS1123(name string) string {
	if len(validation.IsDNS1123Label(name)) != 0 {
		// Use - for any of the non-matching characters
		n := dns1123LabelRegexp.ReplaceAllString(name, "-")

		// Now let's remove any leading or trailing -
		return strings.ToLower(strings.Trim(n, "-"))
	}
	return name
}

// TrimDNS1123Label trims a label to meet the DNS 1123 label length requirement
// by removing characters from the beginning of label such that len(label) <= 63.
func TrimDNS1123Label(label string) string {
	if len(label) > validation.DNS1123LabelMaxLength {
		return strings.Trim(label[len(label)-validation.DNS1123LabelMaxLength:], "-")
	}
	return label
}

// SupportsOwnerReference checks whether a given dependent supports owner references, based on the owner.
// The namespace of the dependent resource can either be passed in explicitly, otherwise it will be
// extracted from the dependent runtime.Object.
// This function performs following checks:
//  -- True: Owner is cluster-scoped.
//  -- True: Both Owner and dependent are Namespaced with in same namespace.
//  -- False: Owner is Namespaced and dependent is Cluster-scoped.
//  -- False: Both Owner and dependent are Namespaced with different namespaces.
func SupportsOwnerReference(restMapper meta.RESTMapper, owner, dependent runtime.Object, depNamespace string) (bool, error) {
	ownerGVK := owner.GetObjectKind().GroupVersionKind()
	ownerMapping, err := restMapper.RESTMapping(ownerGVK.GroupKind(), ownerGVK.Version)
	if err != nil {
		return false, err
	}
	mOwner, err := meta.Accessor(owner)
	if err != nil {
		return false, err
	}

	depGVK := dependent.GetObjectKind().GroupVersionKind()
	depMapping, err := restMapper.RESTMapping(depGVK.GroupKind(), depGVK.Version)
	if err != nil {
		return false, err
	}
	mDep, err := meta.Accessor(dependent)
	if err != nil {
		return false, err
	}
	ownerClusterScoped := ownerMapping.Scope.Name() == meta.RESTScopeNameRoot
	ownerNamespace := mOwner.GetNamespace()
	depClusterScoped := depMapping.Scope.Name() == meta.RESTScopeNameRoot
	if depNamespace == "" {
		depNamespace = mDep.GetNamespace()
	}

	if ownerClusterScoped {
		return true, nil
	}

	if depClusterScoped {
		return false, nil
	}

	if ownerNamespace != depNamespace {
		return false, nil
	}
	// Both owner and dependent are namespace-scoped and in the same namespace.
	return true, nil
}
