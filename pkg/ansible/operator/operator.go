package operator

import (
	"log"
	"math/rand"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/sirupsen/logrus"
)

func RunSDK(done chan error, mgr manager.Manager) {
	namespace := "default"
	watches, err := runner.NewFromWatches("./watches.yaml")
	if err != nil {
		logrus.Error("Failed to get watches")
		done <- err
		return
	}
	rand.Seed(time.Now().Unix())
	c := signals.SetupSignalHandler()

	for gvk, runner := range watches {
		controller.Add(mgr, controller.Options{
			GVK:         gvk,
			Namespace:   namespace,
			Runner:      runner,
			StopChannel: c,
		})
	}
	log.Fatal(mgr.Start(c))
	done <- nil
}
