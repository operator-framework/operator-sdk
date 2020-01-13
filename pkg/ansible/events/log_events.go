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

package events

import (
	"errors"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// LogLevel - Levelt for the logging to take place.
type LogLevel int

const (
	// Tasks - only log the high level tasks.
	Tasks LogLevel = iota

	// Everything - log every event.
	Everything

	// Nothing -  this will log nothing.
	Nothing
)

// EventHandler - knows how to handle job events.
type EventHandler interface {
	Handle(string, *unstructured.Unstructured, eventapi.JobEvent)
}

type loggingEventHandler struct {
	LogLevel LogLevel
}

func (l loggingEventHandler) Handle(ident string, u *unstructured.Unstructured, e eventapi.JobEvent) {
	if l.LogLevel == Nothing {
		return
	}

	logger := logf.Log.WithName("logging_event_handler").WithValues(
		"name", u.GetName(),
		"namespace", u.GetNamespace(),
		"gvk", u.GroupVersionKind().String(),
		"event_type", e.Event,
		"job", ident,
	)

	// logger only the following for the 'Tasks' LogLevel
	t, ok := e.EventData["task"]
	if ok {
		setFactAction := e.EventData["task_action"] == eventapi.TaskActionSetFact
		debugAction := e.EventData["task_action"] == eventapi.TaskActionDebug

		if e.Event == eventapi.EventPlaybookOnTaskStart && !setFactAction && !debugAction {
			logger.Info("[playbook task]", "EventData.Name", e.EventData["name"])
			l.logAnsibleStdOut(e)
			return
		}
		if e.Event == eventapi.EventRunnerOnOk && debugAction {
			logger.Info("[playbook debug]", "EventData.TaskArgs", e.EventData["task_args"])
			l.logAnsibleStdOut(e)
			return
		}
		if e.Event == eventapi.EventRunnerOnFailed {
			errKVs := []interface{}{
				"EventData.Task", t,
				"EventData.TaskArgs", e.EventData["task_args"],
			}
			if taskPath, ok := e.EventData["task_path"]; ok {
				errKVs = append(errKVs, "EventData.FailedTaskPath", taskPath)
			}
			logger.Error(errors.New("[playbook task failed]"), "", errKVs...)
			l.logAnsibleStdOut(e)
			return
		}
	}

	// log everything else for the 'Everything' LogLevel
	if l.LogLevel == Everything {
		logger.Info("", "EventData", e.EventData)
	}
}

// logAnsibleStdOut will print in the logs the Ansible Task Output formatted
func (l loggingEventHandler) logAnsibleStdOut(e eventapi.JobEvent) {
	fmt.Printf("\n--------------------------- Ansible Task StdOut -------------------------------\n")
	if e.Event != eventapi.EventPlaybookOnTaskStart {
		fmt.Printf(fmt.Sprintf("\n TASK [%v] ******************************** \n", e.EventData["task"]))
	}
	fmt.Println(e.StdOut)
	fmt.Printf("\n-------------------------------------------------------------------------------\n")
}

// NewLoggingEventHandler - Creates a Logging Event Handler to log events.
func NewLoggingEventHandler(l LogLevel) EventHandler {
	return loggingEventHandler{
		LogLevel: l,
	}
}
