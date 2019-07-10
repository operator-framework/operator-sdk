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

package operator

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	"github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var log = logf.Log.WithName("manager")

// Run - A blocking function which starts a controller-runtime manager
// It starts an Operator by reading in the values in `./watches.yaml`, adds a controller
// to the manager, and finally running the manager.
func Run(done chan error, mgr manager.Manager, f *flags.AnsibleOperatorFlags, cMap *controllermap.ControllerMap) {
	watchesSlice, err := watches.Load(f.WatchesFile)
	if err != nil {
		log.Error(err, "Failed to parse watches")
		done <- err
		return
	}
	rand.Seed(time.Now().Unix())
	c := signals.SetupSignalHandler()

	for _, watch := range watchesSlice {
		runner, err := runner.New(watch)
		if err != nil {
			log.Error(err, "Failed to create runner")
			done <- err
			return
		}

		o := controller.Options{
			GVK:             watch.GroupVersionKind,
			Runner:          runner,
			ManageStatus:    watch.ManageStatus,
			MaxWorkers:      getMaxWorkers(watch.GroupVersionKind, f.MaxWorkers),
			ReconcilePeriod: watch.ReconcilePeriod,
		}

		ctr := controller.Add(mgr, o)
		if ctr == nil {
			done <- errors.New("failed to add controller")
			return
		}
		cMap.Store(o.GVK, &controllermap.Contents{Controller: *ctr,
			WatchDependentResources:     watch.WatchDependentResources,
			WatchClusterScopedResources: watch.WatchClusterScopedResources,
			OwnerWatchMap:               controllermap.NewWatchMap(),
			AnnotationWatchMap:          controllermap.NewWatchMap(),
		})
	}
	done <- mgr.Start(c)
}

// if the WORKER_* environment variable is set, use that value.
// Otherwise, use the value from the CLI. This is definitely
// counter-intuitive but it allows the operator admin adjust the
// number of workers based on their cluster resources. While the
// author may use the CLI option to specify a suggested
// configuration for the operator.
func getMaxWorkers(gvk schema.GroupVersionKind, defValue int) int {
	envVar := strings.ToUpper(strings.Replace(
		fmt.Sprintf("WORKER_%s_%s", gvk.Kind, gvk.Group),
		".",
		"_",
		-1,
	))
	switch maxWorkers, err := strconv.Atoi(os.Getenv(envVar)); {
	case maxWorkers <= 1:
		return 1
	case err != nil:
		// we don't care why we couldn't parse it just use default
		log.Info("Failed to parse %v from environment. Using default %v", envVar, defValue)
		return defValue
	default:
		return maxWorkers
	}
}
