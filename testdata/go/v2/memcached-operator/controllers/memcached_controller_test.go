/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/example/memcached-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout   = time.Second * 2
	interval  = time.Millisecond * 200
	namespace = "default"
)

var _ = Describe("MemcachedOperator", func() {

	DescribeTable("we create a memcached resource",
		func(size int32) {
			memcachedName := "memcached-sample-" + fmt.Sprint(size)
			memcached := &v1alpha1.Memcached{
				TypeMeta: metav1.TypeMeta{
					Kind: "memcached",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      memcachedName,
				},
				Spec: v1alpha1.MemcachedSpec{
					Size: size,
				},
			}
			Expect(k8sClient.Create(context.TODO(), memcached)).To(Succeed())

			By(fmt.Sprintf("creating a deployment of size %v", size))
			deploymentLookupKey := types.NamespacedName{
				Namespace: namespace,
				Name:      memcachedName,
			}
			Eventually(func() *int32 {
				deployment := &appsv1.Deployment{}
				k8sClient.Get(context.TODO(), deploymentLookupKey, deployment)

				return deployment.Spec.Replicas
			}, timeout, interval).Should(Equal(&size))
		},
		Entry("of size 1", int32(1)),
		Entry("of size 2", int32(2)),
		Entry("of size 3", int32(3)),
	)
})
