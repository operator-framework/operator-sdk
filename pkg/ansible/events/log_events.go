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
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	// Ansible Events
	EventPlaybookOnTaskStart = "playbook_on_task_start"
	EventRunnerOnOk          = "runner_on_ok"
	EventRunnerOnFailed      = "runner_on_failed"

	// Ansible Task Actions
	TaskActionSetFact = "set_fact"
	TaskActionDebug   = "debug"
)

// EventHandler - knows how to handle job events.
type EventHandler interface {
	Handle(*unstructured.Unstructured, eventapi.JobEvent)
}

type loggingEventHandler struct {
	LogLevel LogLevel
}

func (l loggingEventHandler) Handle(u *unstructured.Unstructured, e eventapi.JobEvent) {
	log := logrus.WithFields(logrus.Fields{
		"component":  "logging_event_handler",
		"name":       u.GetName(),
		"namespace":  u.GetNamespace(),
		"gvk":        u.GroupVersionKind().String(),
		"event_type": e.Event,
	})
	if l.LogLevel == Nothing {
		return
	}
	// log only the following for the 'Tasks' LogLevel
	t, ok := e.EventData["task"]
	if ok {
		setFactAction := e.EventData["task_action"] == TaskActionSetFact
		debugAction := e.EventData["task_action"] == TaskActionDebug

		if e.Event == EventPlaybookOnTaskStart && !setFactAction && !debugAction {
			log.Infof("[playbook task]: %s", e.EventData["name"])
			return
		}
		if e.Event == EventRunnerOnOk && debugAction {
			log.Infof("[playbook debug]: %v", e.EventData["task_args"])
			return
		}
		if e.Event == EventRunnerOnFailed {
			log.Errorf("[failed]: [playbook task] '%s' failed with task_args - %v",
				t, e.EventData["task_args"])
			return
		}
	}
	// log everything else for the 'Everything' LogLevel
	if l.LogLevel == Everything {
		log.Infof("event: %#v", e.EventData)
	}
}

// NewLoggingEventHandler - Creates a Logging Event Handler to log events.
func NewLoggingEventHandler(l LogLevel) EventHandler {
	return loggingEventHandler{
		LogLevel: l,
	}
}
