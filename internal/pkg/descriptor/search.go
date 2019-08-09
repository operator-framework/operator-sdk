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

package descriptor

import (
	"strings"

	"k8s.io/gengo/types"
)

// memberNode is a struct Member containing a pointer to its parent Member,
// if any.
type memberNode struct {
	types.Member
	parentNode *memberNode
}

// descNodeMapping maps a descriptor to its parent Member, if any.
type descNodeMapping struct {
	parentNode *memberNode
	descriptor descriptor
}

// bfsJoinDescriptorPaths performs BFS on all struct members in parentType to
// find members corresponding to descriptors in descMap, which contain the
// member they were parsed from. pt determines parentType;s descriptor type,
// ex. "spec", "status".
func bfsJoinDescriptorPaths(parentType *types.Type, pt descriptorType, descMap map[string][]descriptor) {
	nextMembers, leaves := []*memberNode{}, []descNodeMapping{}
	for _, m := range parentType.Members {
		nextMembers = append(nextMembers, &memberNode{m, nil})
	}
	maxLevel := 10
	level, lenNextMembers := 0, len(nextMembers)
	// BFS up to maxLevel for qualifying leaves. We must check that both the
	// parent type and member type, and member names are equal before adding a
	// leaf. We must loop through all fields in a parent struct type, not just
	// all members in nextMembers, in order to do so. We could find an incorrect
	// leaf if we only checked member type/name equality since different structs
	// can have the same field signatures.
	for len(nextMembers) > 0 && level < maxLevel {
		for _, m := range nextMembers {
			t := getUnderlyingType(m.Type)
			for _, mm := range t.Members {
				node := memberNode{mm, m}
				nextMembers = append(nextMembers, &node)
				mmn := getUnderlyingTypeName(mm.Type)
				if ds, ok := descMap[mmn]; ok {
					newDs := []descriptor{}
					for _, d := range ds {
						typesEqual := typeNamesEqual(m.Type, d.parentType)
						membersEqual := mm.Name == d.member.Name
						if d.descType == pt && typesEqual && membersEqual {
							leaves = append(leaves, descNodeMapping{&node, d})
						} else {
							newDs = append(newDs, d)
						}
					}
					descMap[mmn] = newDs
				}
			}
		}
		nextMembers = nextMembers[lenNextMembers:]
		lenNextMembers = len(nextMembers)
		level++
	}

	seenSpec, seenStatus := false, false
	for _, l := range leaves {
		segments := []string{}
		if l.descriptor.path != "" {
			segments = append(segments, l.descriptor.path)
		}
		for parent := l.parentNode; parent != nil; parent = parent.parentNode {
			pathSeg := ""
			// Use the field's name if it doesn't have a JSON tag.
			if parent.Tags == "" {
				pathSeg = parent.Name
			} else {
				pathSeg = parsePathFromJSONTags(parent.Tags)
			}
			if pathSeg != "" {
				// {Kind}Spec and {Kind}Status pathSegs should not be in the resulting
				// path, as spec/status is implied by specDescriptors/statusDescriptors;
				// children of these types with "spec"/"status" pathSegs should be
				// included.
				if pathSeg == typeSpec && !seenSpec {
					seenSpec = true
					continue
				}
				if pathSeg == typeStatus && !seenStatus {
					seenStatus = true
					continue
				}
				segments = append([]string{pathSeg}, segments...)
			}
		}
		l.descriptor.path = strings.Join(segments, ".")
		n := getUnderlyingTypeName(l.descriptor.member.Type)
		descMap[n] = append(descMap[n], l.descriptor)
	}
}

// getUnderlyingType returns either the Elem or Underlying type of t if t.
func getUnderlyingType(t *types.Type) *types.Type {
	switch t.Kind {
	case types.Map, types.Slice, types.Pointer, types.Chan:
		t = t.Elem
	case types.Alias, types.DeclarationOf:
		t = t.Underlying
	}
	return t
}

// getUnderlyingTypeName stringifies t's underlying type.
func getUnderlyingTypeName(t *types.Type) string {
	return getUnderlyingType(t).Name.String()
}

// typeNamesEqual checks if t1 and t2's underlying type names are equal.
func typeNamesEqual(t1, t2 *types.Type) bool {
	return getUnderlyingTypeName(t1) == getUnderlyingTypeName(t2)
}
