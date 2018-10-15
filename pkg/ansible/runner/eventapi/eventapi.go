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

package eventapi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EventReceiver serves the event API
type EventReceiver struct {
	// Events is the channel used by the event API handler to send JobEvents
	// back to the runner, or whatever code is using this receiver.
	Events chan JobEvent

	// SocketPath is the path on the filesystem to a unix streaming socket
	SocketPath string

	// URLPath is the path portion of the url at which events should be
	// received. For example, "/events/"
	URLPath string

	// server is the http.Server instance that serves the event API. It must be
	// closed.
	server io.Closer

	// stopped indicates if this receiver has permanently stopped receiving
	// events. When true, requests to POST an event will receive a "410 Gone"
	// response, and the body will be ignored.
	stopped bool

	// mutex controls access to the "stopped" bool above, ensuring that writes
	// are goroutine-safe.
	mutex sync.RWMutex

	// ident is the unique identifier for a particular run of ansible-runner
	ident string

	// logger holds a logger that has some fields already set
	logger logrus.FieldLogger
}

func New(ident string, errChan chan<- error) (*EventReceiver, error) {
	sockPath := fmt.Sprintf("/tmp/ansibleoperator-%s", ident)
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	rec := EventReceiver{
		Events:     make(chan JobEvent, 1000),
		SocketPath: sockPath,
		URLPath:    "/events/",
		ident:      ident,
		logger: logrus.WithFields(logrus.Fields{
			"component": "eventapi",
			"job":       ident,
		}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(rec.URLPath, rec.handleEvents)
	srv := http.Server{Handler: mux}
	rec.server = &srv

	go func() {
		errChan <- srv.Serve(listener)
	}()
	return &rec, nil
}

// Close ensures that appropriate resources are cleaned up, such as any unix
// streaming socket that may be in use. Close must be called.
func (e *EventReceiver) Close() {
	e.mutex.Lock()
	e.stopped = true
	e.mutex.Unlock()
	e.logger.Debug("event API stopped")
	e.server.Close()
	close(e.Events)
}

func (e *EventReceiver) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != e.URLPath {
		http.NotFound(w, r)
		e.logger.WithFields(logrus.Fields{
			"code": "404",
		}).Infof("path not found: %s\n", r.URL.Path)
		return
	}

	if r.Method != http.MethodPost {
		e.logger.WithFields(logrus.Fields{
			"code": "405",
		}).Infof("method %s not allowed", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ct := r.Header.Get("content-type")
	if strings.Split(ct, ";")[0] != "application/json" {
		e.logger.WithFields(logrus.Fields{
			"code": "415",
		}).Infof("wrong content type: %s", ct)
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte("The content-type must be \"application/json\""))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"code": "500",
		}).Errorf("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	event := JobEvent{}
	err = json.Unmarshal(body, &event)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"code": "400",
		}).Infof("could not deserialize body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Could not deserialize body as JSON"))
		return
	}

	// Guarantee that the Events channel will not be written to if stopped ==
	// true, because in that case the channel has been closed.
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.stopped {
		e.mutex.RUnlock()
		w.WriteHeader(http.StatusGone)
		e.logger.WithFields(logrus.Fields{
			"code": "410",
		}).Info("stopped and not accepting additional events for this job")
		return
	}
	// ansible-runner sends "status events" and "ansible events". The "status
	// events" signify a change in the state of ansible-runner itself, which
	// we're not currently interested in.
	// https://ansible-runner.readthedocs.io/en/latest/external_interface.html#event-structure
	if event.UUID == "" {
		e.logger.Info("dropping event that is not a JobEvent")
	} else {
		// timeout if the channel blocks for too long
		timeout := time.NewTimer(10 * time.Second)
		select {
		case e.Events <- event:
		case <-timeout.C:
			e.logger.WithFields(logrus.Fields{
				"code": "500",
			}).Warn("timed out writing event to channel")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = timeout.Stop()
	}
	w.WriteHeader(http.StatusNoContent)
}
