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

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/pkg/errors"
	"k8s.io/gengo/types"
)

type typeTree interface {
	getDescriptorsFor(descriptorType) ([]descriptor, error)
}

type ttree struct {
	root      *types.Type
	annotated []*tnode
}

type tnode struct {
	member       types.Member
	children     []*tnode
	pathSegments []string
}

// newTypeTreeFromRoot collects all struct members in root and stores them in
// an ttree, along with any members that have annotations.
func newTypeTreeFromRoot(root *types.Type) (typeTree, error) {
	tree := ttree{root: root}
	nextChildren := []*tnode{{member: types.Member{Type: root}}}
	lenNextChildren := len(nextChildren)
	for len(nextChildren) > 0 {
		for _, child := range nextChildren {
			ct := getUnderlyingType(child.member.Type)
			for _, cm := range ct.Members {
				node := &tnode{member: cm}
				// Parse path here so we can re-construct the path hierarchy later.
				path, err := getPathFromMember(cm)
				if err != nil {
					return nil, errors.Wrapf(err, "error parsing %s type member %s JSON tags", child.member.Type.Name, cm.Name)
				}
				node.pathSegments = append(child.pathSegments, path)
				if hasAnnotations(cm) {
					tree.annotated = append(tree.annotated, node)
				}
				child.children = append(child.children, node)
				nextChildren = append(nextChildren, node)
			}
		}
		nextChildren = nextChildren[lenNextChildren:]
		lenNextChildren = len(nextChildren)
	}
	return &tree, nil
}

// getDescriptorsFor returns descriptors for each annotated type in tree
// for a given descriptorType by parsing annotations on each type member.
func (tree *ttree) getDescriptorsFor(descType descriptorType) (descriptors []descriptor, err error) {
	for _, node := range tree.annotated {
		parsedDescriptors, err := parseCSVGenAnnotations(node.member.CommentLines)
		if err != nil {
			return nil, err
		}
		for _, d := range parsedDescriptors.descriptors {
			if d.include && d.descType == descType {
				pathBuilder := &strings.Builder{}
				var hasIgnore, hasInline, includeInlined bool
				lastIdx := len(node.pathSegments) - 1
				for segmentIdx, segment := range node.pathSegments {
					// Ignored members are not serialized and therefore its own tag and
					// all children should not be included in the final path.
					if isPathIgnore(segment) {
						hasIgnore = true
						break
					}
					// Inlined members move their fields into the parent if the inlined
					// member is not a leaf. This condition prevents inlined annotated
					// members from having incorrect paths.
					if isPathInline(segment) {
						hasInline = true
						includeInlined = segmentIdx < lastIdx || includeInlined
						continue
					}
					pathBuilder.WriteString(segment)
					pathBuilder.WriteString(".")
				}
				if hasIgnore || (hasInline && !includeInlined) {
					continue
				}
				d.Path = strings.Trim(pathBuilder.String(), ".")
				if d.DisplayName == "" {
					d.DisplayName = k8sutil.GetDisplayName(node.member.Name)
				}
				d.Description = parseDescription(node.member.CommentLines)
				switch d.descType {
				case typeSpec:
					d.XDescriptors = getSpecXDescriptorsByPath(d.XDescriptors, d.Path)
				case typeStatus:
					d.XDescriptors = getStatusXDescriptorsByPath(d.XDescriptors, d.Path)
				}
				descriptors = append(descriptors, d)
			}
		}
	}
	return sortDescriptors(descriptors), nil
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

func hasAnnotations(m types.Member) bool {
	return len(types.ExtractCommentTags(csvgenPrefix, m.CommentLines)) != 0
}
