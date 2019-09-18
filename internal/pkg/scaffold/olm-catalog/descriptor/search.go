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

	"k8s.io/gengo/types"
)

type typeTree interface {
	getDescriptors() ([]descriptor, error)
}

type atree struct {
	root      *types.Type
	children  []*anode
	annotated []*anode
}

var _ typeTree = &atree{}

type anode struct {
	member       types.Member
	children     []*anode
	pathSegments []string
}

// newTypeTreeFromRoot collects all struct members in root and stores them in
// an atree, along with any members that have annotations.
func newTypeTreeFromRoot(root *types.Type) typeTree {
	tree := atree{root: root}
	for _, m := range root.Members {
		if !isMetadata(m) {
			tree.children = append(tree.children, &anode{member: m})
		}
	}
	if len(tree.children) == 0 {
		return &tree
	}
	nextChildren := make([]*anode, len(tree.children))
	copy(nextChildren, tree.children)
	lenNextChildren := len(nextChildren)
	for len(nextChildren) > 0 {
		for _, child := range nextChildren {
			ct := getUnderlyingType(child.member.Type)
			for _, cm := range ct.Members {
				node := &anode{member: cm}
				path := parsePathFromJSONTags(cm.Tags)
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
	return &tree
}

func (tree *atree) getDescriptors() (descriptors []descriptor, err error) {
	for _, node := range tree.annotated {
		parsedDescriptors, err := parseCSVGenAnnotations(node.member.CommentLines)
		if err != nil {
			return nil, err
		}
		for _, d := range parsedDescriptors.descriptors {
			d.description = parseDescription(node.member.CommentLines)
			d.displayName = k8sutil.GetDisplayName(node.member.Name)
			d.path = strings.Join(node.pathSegments, ".")
			switch d.descType {
			case typeSpec:
				d.xdescs = getSpecXDescriptorsByPath(d.xdescs, d.path)
			case typeStatus:
				d.xdescs = getStatusXDescriptorsByPath(d.xdescs, d.path)
			}
			descriptors = append(descriptors, d)
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

func isMetadata(m types.Member) bool {
	typeName := m.Type.Name.Name
	return strings.HasSuffix(typeName, "ObjectMeta") || strings.HasSuffix(typeName, "TypeMeta")
}

func hasAnnotations(m types.Member) bool {
	return len(types.ExtractCommentTags(csvgenPrefix, m.CommentLines)) != 0
}
