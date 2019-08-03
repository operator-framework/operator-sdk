// Copyright 2019 The Operator-SDK Authors
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
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// VersionRegexp matches Kubernetes API versions.
	// See https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
	VersionRegexp = regexp.MustCompile("^(v[1-9][0-9]*)((alpha|beta)([1-9][0-9]*))?$")
	// KindRegexp matches Kubernetes API Kind's.
	KindRegexp = regexp.MustCompile("^[A-Z]{1}[a-zA-Z0-9]+$")
)

type versionSort struct {
	len  int
	less func(int, int) bool
	swap func(int, int)
}

func (v versionSort) Len() int           { return v.len }
func (v versionSort) Less(i, j int) bool { return v.less(i, j) }
func (v versionSort) Swap(i, j int)      { v.swap(i, j) }

func runSort(versions interface{}, less func(int, int) bool) {
	swap := reflect.Swapper(versions)
	rv := reflect.ValueOf(versions)
	sorter := versionSort{rv.Len(), less, swap}
	sort.Sort(sorter)
}

// SortVersions sorts CRD versions using the algorithm described in
// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#version-priority
func SortVersions(versions interface{}, verGetter func(interface{}, int) string) {
	less := sortVersionsK8sFunc(versions, verGetter)
	runSort(versions, less)
}

func sortVersionsK8sFunc(versions interface{}, verGetter func(interface{}, int) string) func(int, int) bool {
	return func(i int, j int) bool {
		verI, verJ := verGetter(versions, i), verGetter(versions, j)
		if !VersionRegexp.MatchString(verI) && !VersionRegexp.MatchString(verJ) {
			return verI < verJ
		}
		if !VersionRegexp.MatchString(verI) {
			return false
		}
		if !VersionRegexp.MatchString(verJ) {
			return true
		}
		subsI := VersionRegexp.FindStringSubmatch(verI)
		subsJ := VersionRegexp.FindStringSubmatch(verJ)
		subsI, subsJ = resize(subsI), resize(subsJ)
		// Two single grouping, ex. "(v1)" vs "(v2)"
		if len(subsI) == 1 && len(subsI) == len(subsJ) {
			return parseInt(subsI[0]) > parseInt(subsJ[0])
		}
		// Two quad groupings, ex. "(v1)((alpha)(1))" vs "(v2)((alpha)(1))"
		if len(subsI) == 4 && len(subsI) == len(subsJ) {
			// If both alpha or beta, sort by version numbers.
			if subsI[2] == subsJ[2] {
				if subsI[0] == subsJ[0] {
					return parseInt(subsI[3]) > parseInt(subsJ[3])
				}
				return parseInt(subsI[0]) > parseInt(subsJ[0])
			}
			// If I isn't beta then J must be.
			return subsI[2] == "beta"
		}
		// A single and a quad grouping. Always sort single before quad.
		return len(subsI) == 1
	}
}

// SortVersionsMajor sorts CRD versions by their leading major version.
// Example:
//	input:  ["v1alpha1", "v1", "v2beta4", "v2alpha4", "v3alpha1", "v2"]
//	output: ["v3alpha1", "v2", "v2beta4", "v2alpha4", "v1", "v1alpha1"]
func SortVersionsMajor(versions interface{}, verGetter func(interface{}, int) string) {
	less := sortVersionsMajorFunc(versions, verGetter)
	runSort(versions, less)
}

func sortVersionsMajorFunc(versions interface{}, verGetter func(interface{}, int) string) func(int, int) bool {
	return func(i int, j int) bool {
		verI, verJ := verGetter(versions, i), verGetter(versions, j)
		if !VersionRegexp.MatchString(verI) && !VersionRegexp.MatchString(verJ) {
			return verI < verJ
		}
		if !VersionRegexp.MatchString(verI) {
			return false
		}
		if !VersionRegexp.MatchString(verJ) {
			return true
		}
		subsI := VersionRegexp.FindStringSubmatch(verI)
		subsJ := VersionRegexp.FindStringSubmatch(verJ)
		subsI, subsJ = resize(subsI), resize(subsJ)
		// Two single grouping, ex. "(v1)" vs "(v2)"
		if len(subsI) == 1 && len(subsI) == len(subsJ) {
			return subsI[0] > subsJ[0]
		}
		// Two quad groupings, ex. "(v1)((alpha)(1))" vs "(v2)((alpha)(1))"
		if len(subsI) == 4 && len(subsI) == len(subsJ) {
			if subsI[0] == subsJ[0] {
				if subsI[2] == subsJ[2] {
					return parseInt(subsI[3]) > parseInt(subsJ[3])
				}
				// If I isn't beta then J must be.
				return subsI[2] == "beta"
			}
			return parseInt(subsI[0]) > parseInt(subsJ[0])
		}
		// A single and a quad grouping. Either I and J have equal major groups
		// and I is stable, visa versa, or one has a greater major group.
		firstGroupEqual := subsI[0] == subsJ[0]
		if len(subsI) == 1 {
			return firstGroupEqual || parseInt(subsI[0]) > parseInt(subsJ[0])
		}
		return !firstGroupEqual && parseInt(subsI[0]) >= parseInt(subsJ[0])
	}
}

func GetCRDVersionsName(v interface{}, i int) string {
	versions, ok := v.([]apiextv1beta1.CustomResourceDefinitionVersion)
	if !ok {
		panic("v not a CRD version slice")
	}
	return versions[i].Name
}

func GetStringSliceName(v interface{}, i int) string {
	versions, ok := v.([]string)
	if !ok {
		panic("v not a string slice")
	}
	return versions[i]
}

func resize(subs []string) []string {
	if subs[0] == subs[1] {
		subs = subs[0:1]
	} else {
		subs = subs[1:len(subs)]
	}
	return subs
}

func parseInt(s string) int {
	s = strings.TrimPrefix(s, "v")
	i, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		panic(fmt.Sprintf("%s not an int: %v", s, err))
	}
	return int(i)
}
