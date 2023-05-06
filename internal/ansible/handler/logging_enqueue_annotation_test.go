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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-lib/handler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("LoggingEnqueueRequestForAnnotation", func() {
	var q workqueue.RateLimitingInterface
	var instance LoggingEnqueueRequestForAnnotation
	var pod *corev1.Pod
	var podOwner *corev1.Pod

	BeforeEach(func() {
		q = controllertest.Queue{Interface: workqueue.New()}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "biz",
				Name:      "biz",
			},
		}
		podOwner = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "podOwnerNs",
				Name:      "podOwnerName",
			},
		}

		pod.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
		podOwner.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})

		Expect(handler.SetOwnerAnnotations(podOwner, pod)).To(Succeed())
		instance = LoggingEnqueueRequestForAnnotation{
			handler.EnqueueRequestForAnnotation{
				Type: schema.GroupKind{
					Group: "",
					Kind:  "Pod",
				}}}
	})

	Describe("Create", func() {
		It("should emit a log and enqueue a Request with the annotations of the object in case of CreateEvent", func() {
			evt := event.CreateEvent{
				Object: pod,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})

		It("should enqueue a Request to the owner resource when the annotations are applied in child object"+
			" in the Create Event", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			Expect(handler.SetOwnerAnnotations(podOwner, repl)).To(Succeed())

			evt := event.CreateEvent{
				Object: repl,
			}
			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*apps/v1.*ReplicaSet.*faz.*foo.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})
		It("should not emit a log or enqueue a request if there are no annotations matching with the object", func() {
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
		It("should not emit a log or enqueue a Request if there is no Namespace and name annotation matching the specified object are found", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						handler.TypeAnnotation: schema.GroupKind{Group: "", Kind: "Pod"}.String(),
					},
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
		It("should not emit a log or enqueue a Request if there is no TypeAnnotation matching the specified Group and Kind", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",

					Annotations: map[string]string{
						handler.NamespacedNameAnnotation: "AppService",
					},
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
		It("should emit a log and enqueue a Request if there are no Namespace annotation matching the object", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
					Annotations: map[string]string{
						handler.NamespacedNameAnnotation: "AppService",
						handler.TypeAnnotation:           schema.GroupKind{Group: "", Kind: "Pod"}.String(),
					},
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*apps/v1.*ReplicaSet.*faz.*foo.*Pod.*AppService`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "AppService"}}))
		})
		It("should emit a log and enqueue a Request for an object that is cluster scoped which has the annotations", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Annotations: map[string]string{
						handler.NamespacedNameAnnotation: "myapp",
						handler.TypeAnnotation:           schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}.String(),
					},
				},
			}
			nd.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"})

			instance = LoggingEnqueueRequestForAnnotation{handler.EnqueueRequestForAnnotation{Type: schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}}}

			evt := event.CreateEvent{
				Object: nd,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Create.*/v1.*Node.*node-1.*ReplicaSet.apps.*myapp.*`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "", Name: "myapp"}}))
		})
		It("should not emit a log or enqueue a Request for an object that is cluster scoped which does not have annotations", func() {
			nd := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			}
			nd.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"})

			instance = LoggingEnqueueRequestForAnnotation{handler.EnqueueRequestForAnnotation{Type: nd.GetObjectKind().GroupVersionKind().GroupKind()}}
			evt := event.CreateEvent{
				Object: nd,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))
			Expect(q.Len()).To(Equal(0))
		})
	})

	Describe("Delete", func() {
		It("should emit a log and enqueue a Request with the annotations of the object in case of DeleteEvent", func() {
			evt := event.DeleteEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Delete(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Delete.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})
	})

	Describe("Update", func() {
		It("should emit a log and enqueue a Request with annotations applied to both objects in UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			Expect(handler.SetOwnerAnnotations(podOwner, pod)).To(Succeed())

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}

			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})
		It("should emit a log and enqueue a Request with the annotations applied in one of the objects in case of UpdateEvent", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"
			newPod.Annotations = map[string]string{}

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}

			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))
			i, _ := q.Get()

			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})
		It("should emit a log and enqueue a Request when the annotations are applied in a different resource in case of UpdateEvent", func() {
			repl := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "faz",
				},
			}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})

			instance = LoggingEnqueueRequestForAnnotation{
				handler.EnqueueRequestForAnnotation{
					Type: schema.GroupKind{
						Group: "apps",
						Kind:  "ReplicaSet",
					}}}

			evt := event.CreateEvent{
				Object: repl,
			}

			logBuffer.Reset()
			instance.Create(evt, q)
			Expect(logBuffer.String()).To(Not(ContainSubstring("ansible.handler")))
			Expect(q.Len()).To(Equal(0))

			newRepl := repl.DeepCopy()
			newRepl.Name = pod.Name + "2"
			newRepl.Namespace = pod.Namespace + "2"

			newRepl.Annotations = map[string]string{
				handler.TypeAnnotation:           schema.GroupKind{Group: "apps", Kind: "ReplicaSet"}.String(),
				handler.NamespacedNameAnnotation: "foo/faz",
			}

			instance2 := LoggingEnqueueRequestForAnnotation{
				handler.EnqueueRequestForAnnotation{
					Type: schema.GroupKind{
						Group: "apps",
						Kind:  "ReplicaSet",
					}}}

			evt2 := event.UpdateEvent{
				ObjectOld: repl,
				ObjectNew: newRepl,
			}

			logBuffer.Reset()
			instance2.Update(evt2, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*apps/v1.*ReplicaSet.*faz.*foo.*ReplicaSet.apps.*faz.*foo`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "foo",
					Name:      "faz",
				},
			}))
		})
		It("should emit a log and enqueue multiple Update Requests when different annotations are applied to multiple objects", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			Expect(handler.SetOwnerAnnotations(podOwner, pod)).To(Succeed())

			var podOwner2 = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "podOwnerNsTest",
					Name:      "podOwnerNameTest",
				},
			}
			podOwner2.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Kind: "Pod"})

			Expect(handler.SetOwnerAnnotations(podOwner2, newPod)).To(Succeed())

			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}
			logBuffer.Reset()
			instance.Update(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Update.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(2))
		})
	})

	Describe("Generic", func() {
		It("should enqueue a Request with the annotations of the object in case of GenericEvent", func() {
			evt := event.GenericEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Generic(evt, q)
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*Generic.*/v1.*Pod.*biz.*biz.*Pod.*podOwnerName.*podOwnerNs`,
			))
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: podOwner.Namespace,
					Name:      podOwner.Name,
				},
			}))
		})
	})
})
