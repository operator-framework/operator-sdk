/*
	Copyright 2018 The Kubernetes Authors.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

			http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package handler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Eventhandler", func() {
	var ctx = context.Background()
	var q workqueue.RateLimitingInterface
	var pod *corev1.Pod
	var mapper meta.RESTMapper
	BeforeEach(func() {
		q = &controllertest.Queue{Interface: workqueue.New()}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "biz", Name: "baz"},
		}
		pod.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
		Expect(cfg).NotTo(BeNil())

		httpClient, err := rest.HTTPClientFor(cfg)
		Expect(err).ShouldNot(HaveOccurred())
		mapper, err = apiutil.NewDiscoveryRESTMapper(cfg, httpClient)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("EnqueueRequestForOwnerWithLogging", func() {
		var repl *appsv1.ReplicaSet

		BeforeEach(func() {
			repl = &appsv1.ReplicaSet{}
			repl.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})
		})

		It("should enqueue a Request with the Owner of the object in the CreateEvent.", func() {
			instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, repl)

			pod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			evt := event.CreateEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Create(ctx, evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pod.GetNamespace(), Name: "foo-parent"}}))
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*OwnerReference.*handler.*event.*Create.*Owner.APIVersion.*Owner.Kind.*Owner.Name.*`,
			))
		})

		It("should enqueue a Request with the Owner of the object in the DeleteEvent.", func() {
			instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, repl)
			pod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			evt := event.DeleteEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Delete(ctx, evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pod.GetNamespace(), Name: "foo-parent"}}))
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*OwnerReference.*handler.*event.*Delete.*Owner.APIVersion.*Owner.Kind.*Owner.Name.*`,
			))
		})

		It("should enqueue a Request with the Owners of both objects in the UpdateEvent.", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"
			newPod.Namespace = pod.Namespace + "2"

			instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, repl)

			pod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo1-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			newPod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo2-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			logBuffer.Reset()
			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}
			instance.Update(ctx, evt, q)
			Expect(q.Len()).To(Equal(2))

			i1, _ := q.Get()
			i2, _ := q.Get()
			Expect([]interface{}{i1, i2}).To(ConsistOf(
				reconcile.Request{
					NamespacedName: types.NamespacedName{Namespace: pod.GetNamespace(), Name: "foo1-parent"}},
				reconcile.Request{
					NamespacedName: types.NamespacedName{Namespace: newPod.GetNamespace(), Name: "foo2-parent"}},
			))
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*OwnerReference.*handler.*event.*Update.*Owner.APIVersion.*Owner.Kind.*Owner.Name.*`,
			))
		})

		It("should enqueue a Request with the one duplicate Owner of both objects in the UpdateEvent.", func() {
			newPod := pod.DeepCopy()
			newPod.Name = pod.Name + "2"

			instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, &appsv1.ReplicaSet{})

			pod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			newPod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			evt := event.UpdateEvent{
				ObjectOld: pod,
				ObjectNew: newPod,
			}
			instance.Update(ctx, evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pod.GetNamespace(), Name: "foo-parent"}}))
		})

		It("should enqueue a Request with the Owner of the object in the GenericEvent.", func() {
			instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, repl)
			pod.OwnerReferences = []metav1.OwnerReference{
				{
					Name:       "foo-parent",
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
			}
			evt := event.GenericEvent{
				Object: pod,
			}
			logBuffer.Reset()
			instance.Generic(ctx, evt, q)
			Expect(q.Len()).To(Equal(1))

			i, _ := q.Get()
			Expect(i).To(Equal(reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: pod.GetNamespace(), Name: "foo-parent"}}))
			Expect(logBuffer.String()).To(MatchRegexp(
				`ansible.handler.*OwnerReference.*handler.*event.*Generic.*Owner.APIVersion.*Owner.Kind.*Owner.Name.*`,
			))
		})

		Context("with a nil object", func() {
			It("should do nothing.", func() {
				instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, &appsv1.ReplicaSet{})
				pod.OwnerReferences = []metav1.OwnerReference{
					{
						Name:       "foo1-parent",
						Kind:       "ReplicaSet",
						APIVersion: "apps/v1",
					},
				}
				evt := event.CreateEvent{
					Object: nil,
				}
				instance.Create(ctx, evt, q)
				Expect(q.Len()).To(Equal(0))
			})
		})

		Context("with a nil OwnerType", func() {
			It("should panic", func() {
				Expect(func() {
					EnqueueRequestForOwnerWithLogging(nil, nil, nil)
				}).To(Panic())
			})
		})

		Context("with an invalid APIVersion in the OwnerReference", func() {
			It("should do nothing.", func() {
				instance := EnqueueRequestForOwnerWithLogging(scheme.Scheme, mapper, &appsv1.ReplicaSet{})
				pod.OwnerReferences = []metav1.OwnerReference{
					{
						Name:       "foo1-parent",
						Kind:       "ReplicaSet",
						APIVersion: "apps/v1/fail",
					},
				}
				evt := event.CreateEvent{
					Object: pod,
				}
				instance.Create(ctx, evt, q)
				Expect(q.Len()).To(Equal(0))
			})
		})
	})

})
