// Copyright 2021 The Operator-SDK Authors
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

package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crHandler "sigs.k8s.io/controller-runtime/pkg/handler"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("LoggingEnqueueRequestForOwner", func() {
	var q workqueue.RateLimitingInterface
	var instance LoggingEnqueueRequestForOwner
	var pod *corev1.Pod
	var podOwner *metav1.OwnerReference

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		podOwner = &metav1.OwnerReference{
			Kind:       "Pod",
			APIVersion: "v1",
			Name:       "podOwnerName",
		}

		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "biz",
				Name:            "biz",
				OwnerReferences: []metav1.OwnerReference{*podOwner},
			},
		}

		pod.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})

		instance = LoggingEnqueueRequestForOwner{
			crHandler.EnqueueRequestForOwner{
				OwnerType: pod,
			}}
	})

	Describe("Create", func() {
		It("should emit a log with the ownerReference of the object in case of CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*/v1.*Pod.*biz.*biz.*v1.*Pod.*podOwnerName`,
			))
		})

		It("emit a log when the ownerReferences are applied in child object"+
			" in the Create Event", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "foo",
					Name:            "faz",
					OwnerReferences: []metav1.OwnerReference{*podOwner},
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			evt := event.CreateEvent{
				Object: repl,
			}
			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*apps/v1.*ReplicaSet.*faz.*foo.*Pod.*podOwnerName`,
			))
		})
		It("should not emit a log or there are no ownerReferences matching with the object", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))
			Expect(q.Len()).To(Equal(0))
		})
		It("should not emit a log if the ownerReference does not match the OwnerType", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						Name:       "podOwnerName",
					}},
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))
		})

		It("should not emit a log for an object which does not have ownerReferences", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))
		})
	})

	Describe("Delete", func() {
		It("should emit a log with the ownerReferenc of the object in case of DeleteEvent", func() {
			evt := event.DeleteEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Delete(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Delete.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName`,
			))
		})
	})

	Describe("Update", func() {
		It("should emit a log and enqueue a Request with annotations applied to both objects in UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}

			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName`,
			))
		})
		It("should emit a log with the ownerReferences applied in one of the objects in case of UpdateEvent", func() {
			noOwnerPod := pod.DeepCopy()
			noOwnerPod.Name = pod.Name + "2"
			noOwnerPod.Namespace = pod.Namespace + "2"
			noOwnerPod.OwnerReferences = []metav1.OwnerReference{}

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: noOwnerPod,
			}

			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName`,
			))

			evt = event.UpdateEvent{
				ObjectOld: noOwnerPod,
				ObjectNew: pod,
			}

			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName`,
			))
		})
		It("should emit a log when the OwnerReference is applied after creation in case of UpdateEvent", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			instance = LoggingEnqueueRequestForOwner{
				crHandler.EnqueueRequestForOwner{
					OwnerType: repl,
				}}

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))

			newRepl := repl.DeepCopy()
			newRepl.Name = pod.Name + "2"
			newRepl.Namespace = pod.Namespace + "2"

			newRepl.OwnerReferences = []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "faz",
			}}

			evt2 := event.UpdateEvent{
				ObjectOld: repl,
				ObjectNew: newRepl,
			}

			logBuffer.Reset()
			instance.Update(evt2, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*apps/v1.*ReplicaSet.*faz.*foo.*apps/v1.*ReplicaSet.*faz`,
			))
		})
	})

	Describe("Generic", func() {
		It("should emit a log with the OwnerReference of the object in case of GenericEvent", func() {
			evt := event.GenericEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Generic(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Generic.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName`,
			))
		})
	})
})
