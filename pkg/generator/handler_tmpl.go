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

package generator

// handlerTmpl is the template for stub/handler.go.
const handlerTmpl = `package stub

import (
	"{{.OperatorSDKImport}}/handler"
	"{{.OperatorSDKImport}}/types"
	"github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) []types.Action {
	// Change me
	switch o := event.Object.(type) {
	case *apps_v1.Deployment:
		logrus.Printf("Received Deployment: %v", o.Name)
	}
	return nil
}
`
