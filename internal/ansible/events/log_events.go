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
	"os"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/operator-framework/operator-sdk/internal/ansible/runner/eventapi"
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
	mux      *sync.Mutex
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

	verbosity := GetVerbosity(u, e, ident)

	// logger only the following for the 'Tasks' LogLevel
	t, ok := e.EventData["task"]
	if ok {
		setFactAction := e.EventData["task_action"] == eventapi.TaskActionSetFact
		debugAction := e.EventData["task_action"] == eventapi.TaskActionDebug

		if verbosity > 0 {
			l.mux.Lock()
			fmt.Println(e.StdOut)
			l.mux.Unlock()
			return
		}
		if e.Event == eventapi.EventPlaybookOnTaskStart && !setFactAction && !debugAction {
			l.mux.Lock()
			logger.Info("[playbook task start]", "EventData.Name", e.EventData["name"])
			l.logAnsibleStdOut(e)
			l.mux.Unlock()
			return
		}
		if e.Event == eventapi.EventRunnerOnOk && debugAction {
			l.mux.Lock()
			logger.Info("[playbook debug]", "EventData.TaskArgs", e.EventData["task_args"])
			l.logAnsibleStdOut(e)
			l.mux.Unlock()
			return
		}
		if e.Event == eventapi.EventRunnerItemOnOk {
			l.mux.Lock()
			l.logAnsibleStdOut(e)
			l.mux.Unlock()
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
			l.mux.Lock()
			logger.Error(errors.New("[playbook task failed]"), "", errKVs...)
			l.logAnsibleStdOut(e)
			l.mux.Unlock()
			return
		}
	}

	// log everything else for the 'Everything' LogLevel
	if l.LogLevel == Everything {
		l.mux.Lock()
		logger.Info("", "EventData", e.EventData)
		l.logAnsibleStdOut(e)
		l.mux.Unlock()
	}
}

// logAnsibleStdOut will print in the logs the Ansible Task Output formatted
func (l loggingEventHandler) logAnsibleStdOut(e eventapi.JobEvent) {
	if len(e.StdOut) > 0 {
		fmt.Printf("\n--------------------------- Ansible Task StdOut -------------------------------\n")
		if e.Event != eventapi.EventPlaybookOnTaskStart {
			fmt.Printf("\n TASK [%v] ******************************** \n", e.EventData["task"])
		}
		fmt.Println(e.StdOut)
		fmt.Printf("\n-------------------------------------------------------------------------------\n")
	}
}

// NewLoggingEventHandler - Creates a Logging Event Handler to log events.
func NewLoggingEventHandler(l LogLevel) EventHandler {
	return loggingEventHandler{
		LogLevel: l,
		mux:      &sync.Mutex{},
	}
}

// GetVerbosity - Parses the verbsoity from CR and environment variables
func GetVerbosity(u *unstructured.Unstructured, e eventapi.JobEvent, ident string) int {
	logger := logf.Log.WithName("logging_event_handler").WithValues(
		"name", u.GetName(),
		"namespace", u.GetNamespace(),
		"gvk", u.GroupVersionKind().String(),
		"event_type", e.Event,
		"job", ident,
	)

	// Parse verbosity from CR
	verbosityAnnotation := 0
	if annot, exists := u.UnstructuredContent()["metadata"].(map[string]interface{})["annotations"]; exists {
		if verbosityField, present := annot.(map[string]interface{})["ansible.sdk.operatorframework.io/verbosity"]; present {
			var err error
			verbosityAnnotation, err = strconv.Atoi(verbosityField.(string))
			if err != nil {
				logger.Error(err, "Unable to parse verbosity value from CR.")
			}
		}
	}

	// Parse verbosity from environment variable
	verbosityEnvVar := 0
	everb := os.Getenv("ANSIBLE_VERBOSITY")
	if everb != "" {
		var err error
		verbosityEnvVar, err = strconv.Atoi(everb)
		if err != nil {
			logger.Error(err, "Unable to parse verbosity value from environment variable.")
		}
	}

	// Return in order of precedence
	if verbosityAnnotation > 0 {
		return verbosityAnnotation
	} else if verbosityEnvVar > 0 {
		return verbosityEnvVar
	} else {
		return 0 // Default
	}
}
