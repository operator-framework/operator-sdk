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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/input/input.go

package input

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"text/template/parse"
)

// Validator validates input.
type Validator interface {
	// Validate returns nil if the inputs' validation logic approves of
	// field values. Validation of the template is performed by
	// CheckFileTemplateFields and should not be done by Validate().
	Validate() error
}

type ErrEmptyScaffoldField struct {
	Field string
	Value interface{}
}

func (e ErrEmptyScaffoldField) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("template field %s cannot have value %v", e.Field, e.Value)
	}
	return fmt.Sprintf("template field %s not set", e.Field)
}

func CheckFileTemplateFields(f File) (err error) {
	i, err := f.GetInput()
	if err != nil {
		return err
	}
	fields, err := getTemplatePipelines(i)
	if err != nil {
		return err
	}
	if fields == nil {
		return nil
	}

	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Ptr {
		if !v.IsNil() {
			v = v.Elem()
		} else {
			return fmt.Errorf("scaffold %s is nil", v.Type().Name())
		}
	}
	splitFields := map[string][]string{}
	for field := range fields {
		splitFields[field] = strings.Split(field, ".")[1:]
	}
	for _, splitPath := range splitFields {
		fv := v
		pathSoFar := ""
		for _, currPath := range splitPath {
			pathSoFar = strings.Trim(pathSoFar+"."+currPath, ".")
			switch fv.Kind() {
			case reflect.Struct:
				_, found := fv.Type().FieldByName(currPath)
				if !found {
					return fmt.Errorf("template field %s does not exist in struct %s", pathSoFar, fv.Type().Name())
				}
				fv = fv.FieldByName(currPath)
				if isEmptyValue(fv) {
					return ErrEmptyScaffoldField{Field: pathSoFar}
				}
				switch fv.Kind() {
				case reflect.Ptr, reflect.Interface:
					fv = fv.Elem()
				}
			default:
				if isEmptyValue(fv) {
					return ErrEmptyScaffoldField{Field: pathSoFar}
				}
				break
			}
		}
	}
	return nil
}

func IsEmptyValue(i interface{}) bool {
	if v, ok := i.(reflect.Value); ok {
		return isEmptyValue(v)
	}
	return isEmptyValue(reflect.ValueOf(i))
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isEmptyValue(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func getTemplatePipelines(i Input) (map[string]struct{}, error) {
	if i.TemplateBody == "" {
		return nil, nil
	}

	t := template.New("").Funcs(i.TemplateFuncs)
	if i.Delims[0] != "" && i.Delims[1] != "" {
		t = t.Delims(i.Delims[0], i.Delims[1])
	}
	t, err := t.Parse(i.TemplateBody)
	if err != nil {
		return nil, err
	}
	fields := map[string]struct{}{}
	currNodes := t.Tree.Root.Nodes
	currNodesLen := len(currNodes)
	for len(currNodes) > 0 {
		for _, n := range currNodes {
			switch v := n.(type) {
			case *parse.ActionNode:
				currNodes = append(currNodes, v.Pipe)
			case *parse.ListNode:
				currNodes = append(currNodes, v.Nodes...)
			case *parse.PipeNode:
				for _, n := range v.Cmds {
					currNodes = append(currNodes, n)
				}
				for _, n := range v.Decl {
					currNodes = append(currNodes, n)
				}
			case *parse.CommandNode:
				currNodes = append(currNodes, v.Args...)
			case *parse.FieldNode:
				fields[v.String()] = struct{}{}
			case *parse.TemplateNode:
				currNodes = append(currNodes, v.Pipe)
			}
		}
		currNodes = currNodes[currNodesLen:]
		currNodesLen = len(currNodes)
	}

	return fields, nil
}
