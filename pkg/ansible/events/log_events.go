package events

import (
	"github.com/sirupsen/logrus"
	"github.com/water-hole/ansible-operator/pkg/runner/eventapi"
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
	t, ok := e.EventData["task"]
	if ok {
		log = log.WithField("task", t)
	}
	switch l.LogLevel {
	case Everything:
		log.Infof("event: %#v", e.EventData)
	case Tasks:
		if ok {
			log.Infof("event: %#v", e.EventData)
		}
	}
}

// NewLoggingEventHandler - Creates a Logging Event Handler to log events.
func NewLoggingEventHandler(l LogLevel) EventHandler {
	return loggingEventHandler{
		LogLevel: l,
	}
}
