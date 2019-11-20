package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	wh "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = logf.Log.WithName("webhook")

func CreateConversionHook(whConfig watches.WebhookConfig, watch watches.Watch) *wh.Admission {
	return nil
}
func CreateMutatingHook(whConfig watches.WebhookConfig, watch watches.Watch) *wh.Admission {
	return nil
}

func CreateValidatingHook(whConfig watches.WebhookConfig, watch watches.Watch) *wh.Admission {
	r, err := runner.New(watches.Watch{
		GroupVersionKind: watch.GroupVersionKind,
		Role:             whConfig.Role,
		Playbook:         whConfig.Playbook,
	})
	if err != nil {
		// TODO
		panic("aaah")
	}
	return &wh.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req wh.AdmissionRequest) wh.AdmissionResponse {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(watch.GroupVersionKind)
			err = json.Unmarshal(req.Object.Raw, &u)
			if err != nil {
				log.Error(err, "Failed to marshal object")
				return wh.Errored(500, err)
			}

			ownerRef := metav1.OwnerReference{
				APIVersion: u.GetAPIVersion(),
				Kind:       u.GetKind(),
				Name:       u.GetName(),
				UID:        u.GetUID(),
			}
			ident := strconv.Itoa(rand.Int())
			kc, err := kubeconfig.Create(ownerRef, "http://localhost:8888", u.GetNamespace())
			if err != nil {
				log.Error(err, "Failed to create kubeconfig")
				return wh.Errored(500, err)
			}
			result, err := r.Run(ident, u, kc.Name())
			if err != nil {
				log.Error(err, "Run failed")
				return wh.Errored(500, err)
			}
			// iterate events from ansible, looking for the final one
			statusEvent := eventapi.StatusJobEvent{}
			failureMessages := eventapi.FailureMessages{}
			for event := range result.Events() {
				// for _, eHandler := range r.EventHandlers {
				// 	go eHandler.Handle(ident, u, event)
				// }
				if event.Event == eventapi.EventPlaybookOnStats {
					// convert to StatusJobEvent; would love a better way to do this
					data, err := json.Marshal(event)
					if err != nil {
						return wh.Errored(500, err)
					}
					err = json.Unmarshal(data, &statusEvent)
					if err != nil {
						return wh.Errored(500, err)
					}
				}
				if event.Event == eventapi.EventRunnerOnFailed && !event.IgnoreError() {
					failureMessages = append(failureMessages, event.GetFailedPlaybookMessage())
				}
			}
			if statusEvent.Event == "" {
				eventErr := errors.New("did not receive playbook_on_stats event")
				stdout, err := result.Stdout()
				if err != nil {
					log.Error(err, "Failed to get ansible-runner stdout")
					return wh.Errored(500, err)
				}
				log.Error(eventErr, stdout)
				return wh.Errored(500, eventErr)
			}
			log.Info(fmt.Sprintf("%+v", result))
			if len(failureMessages) > 0 {
				return wh.Denied(strings.Join(failureMessages, ", "))
			}
			return wh.Allowed("Ansible run succeeded")
		}),
	}
}
