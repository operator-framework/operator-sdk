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

package main

import (
	"log"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// main renders scaffolds that are required to build the ansible operator base
// image. It is intended for release engineering use only. After running this,
// you can `dep ensure` and then `operator-sdk build`.
func main() {
	cfg := &input.Config{
		AbsProjectPath: projutil.MustGetwd(),
		ProjectName:    "ansible-operator",
	}

	s := &scaffold.Scaffold{}
	err := s.Execute(cfg,
		&ansible.Main{},
		&ansible.GopkgToml{},
		&ansible.DockerfileHybrid{},
		&ansible.Entrypoint{},
		&ansible.UserSetup{},
	)
	if err != nil {
		log.Fatalf("add scaffold failed: (%v)", err)
	}
}
