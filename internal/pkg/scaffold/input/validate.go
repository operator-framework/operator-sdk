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

// Validate validates input
type Validate interface {
	// Validate returns nil if the inputs' validation logic approves of
	// field values. Validation of the template is performed by
	// CheckFileTemplateFields and should not be done by Validate().
	Validate() error
}

type ErrEmptyScaffoldField struct {
	ScaffoldName string
	Field        string
	Value        interface{}
}

func NewEmptyScaffoldFieldError(f File, fieldName string, value ...interface{}) error {
	ft := reflect.TypeOf(f)
	if ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	e := ErrEmptyScaffoldField{
		ScaffoldName: ft.Name(),
		Field:        fieldName,
	}
	if len(value) == 1 {
		e.Value = value[0]
	}
	return e
}

func (e ErrEmptyScaffoldField) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validate %s: template field %s cannot have value %v", e.ScaffoldName, e.Field, e.Value)
	}
	return fmt.Sprintf("validate %s: template field %s not set", e.ScaffoldName, e.Field)
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
			return fmt.Errorf("scaffold %s is nil\n", v.Type().Name())
		}
	}
	splitFields := map[string][]string{}
	for field := range fields {
		splitFields[field] = strings.Split(field, ".")[1:]
	}
	for _, splitPath := range splitFields {
		fv := v
		pathSoFar := ""
		// fmt.Println("split path:", splitPath)
		for _, currPath := range splitPath {
			pathSoFar = strings.Trim(pathSoFar+"."+currPath, ".")
			if fv.Kind() == reflect.Struct {
				fieldValue := fv.FieldByName(currPath)
				// fmt.Printf("\tscaffold %s field %s\n", v.Type().Name(), pathSoFar)
				if isEmptyValue(fieldValue) {
					// fmt.Printf("\t\tempty\n")
					return ErrEmptyScaffoldField{ScaffoldName: v.Type().Name(), Field: pathSoFar}
				} else {
					// fmt.Printf("\t\tnot empty: %v\n", fieldValue)
					switch fieldValue.Kind() {
					case reflect.Struct:
						fv = fieldValue.FieldByName(currPath)
					case reflect.Ptr, reflect.Interface:
						fv = fieldValue.Elem()
					default:
						break
					}
				}
			} else {
				// fmt.Printf("\tscaffold %s non struct field %s\n", v.Type().Name(), pathSoFar)
				if isEmptyValue(fv) {
					// fmt.Printf("\t\tnon struct empty\n")
					return ErrEmptyScaffoldField{ScaffoldName: v.Type().Name(), Field: pathSoFar}
				} else {
					// fmt.Printf("\t\tnon struct not empty: %v\n", fv)
				}
				break
			}
		}
	}
	return nil
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
	case reflect.Bool:
		// Ignore bools since bool being false is valid.
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
				// fmt.Println("Action node: \n", n.String())
				currNodes = append(currNodes, v.Pipe)
			case *parse.ListNode:
				// fmt.Println("List node:", v.String())
				currNodes = append(currNodes, v.Nodes...)
			case *parse.ChainNode:
				// fmt.Println("Chain node:", v.String())
			case *parse.PipeNode:
				// fmt.Println("Pipe node:", v.String())
				for _, n := range v.Cmds {
					currNodes = append(currNodes, n)
				}
				for _, n := range v.Decl {
					currNodes = append(currNodes, n)
				}
			case *parse.BoolNode:
				// fmt.Println("Bool node:", v.String())
			case *parse.BranchNode:
				// fmt.Println("Branch node:", v.String())
				// currNodes = append(currNodes, v.Pipe, v.List)
				// if v.ElseList != nil  {
				// 	currNodes = append(currNodes, v.ElseList)
				// }
				// if v.ElseList == nil || len(v.ElseList.Nodes) == 0 {
				// 	currNodes = append(currNodes, v.Pipe, v.List)
				// }
			case *parse.CommandNode:
				// fmt.Println("Command node:", v.String())
				currNodes = append(currNodes, v.Args...)
			case *parse.DotNode:
				// fmt.Println("Dot node:", v.String())
			case *parse.FieldNode:
				// fmt.Println("Field node:", v.String())
				fields[v.String()] = struct{}{}
			case *parse.IdentifierNode:
				// fmt.Println("Identifier node:", v.String())
			case *parse.IfNode:
				// fmt.Println("If node:", v.String())
				// currNodes = append(currNodes, &v.BranchNode)
			case *parse.NilNode:
				// fmt.Println("Nil node:", v.String())
			case *parse.NumberNode:
				// fmt.Println("Number node:", v.String())
			case *parse.RangeNode:
				// fmt.Println("Range node:", v.String())
				// currNodes = append(currNodes, &v.BranchNode)
			case *parse.StringNode:
				// fmt.Println("String node:", v.String())
			case *parse.TemplateNode:
				// fmt.Println("Template node:", v.String())
				currNodes = append(currNodes, v.Pipe)
			case *parse.VariableNode:
				// fmt.Println("Variable node:", v.String())
			case *parse.WithNode:
				// fmt.Println("With node:", v.String())
				// currNodes = append(currNodes, &v.BranchNode)
			}
		}
		currNodes = currNodes[currNodesLen:]
		currNodesLen = len(currNodes)
	}

	return fields, nil
}
