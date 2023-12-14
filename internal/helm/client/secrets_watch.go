/*
Copyright 2020 The Operator-SDK Authors.

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

package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/selection"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	applyconfv1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("helm.watchedsecrets")

const helmSecretsLabelKey = "owner"
const helmSecretsLabelValue = "helm"

// Wraps the kubernetes SecretInterface
// Helm queries its own secrets multiple times per reconciliation. To reduce the number of lists going to the apiserver
// we instead use an informer to watch the changes on secrets.
type WatchedSecrets struct {
	inner           typedcorev1.SecretInterface
	informerFactory informers.SharedInformerFactory
	informerLister  listerscorev1.SecretNamespaceLister
}

func (w *WatchedSecrets) Create(ctx context.Context, secret *corev1.Secret, opts metav1.CreateOptions) (*corev1.Secret, error) {
	return w.inner.Create(ctx, secret, opts)
}

func (w *WatchedSecrets) Update(ctx context.Context, secret *corev1.Secret, opts metav1.UpdateOptions) (*corev1.Secret, error) {
	return w.inner.Update(ctx, secret, opts)
}

func (w *WatchedSecrets) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return w.inner.Delete(ctx, name, opts)
}

func (w *WatchedSecrets) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return w.inner.DeleteCollection(ctx, opts, listOpts)
}

func (w *WatchedSecrets) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
	return w.inner.Get(ctx, name, opts)
}

func (w *WatchedSecrets) List(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
	labelSelector, err := labels.Parse(opts.LabelSelector)
	if err != nil {
		return nil, err
	}
	ownerLabelSelector, hasOwnerLabelSelector := labelSelector.RequiresExactMatch(helmSecretsLabelKey)

	// The informer interface only offers to filter secrets with a labelSelector
	// We are only watching secrets with label owner=helm.
	// Currently (helm v3.10.3) this List function is only being called in `storage/driver/secrets.go` with a
	// labelSelector, meaning this case should never be executed. But we are able to fallback to the normal List
	// implementation.
	if hasListOptionsOtherThanLabelSelector(opts) || !hasOwnerLabelSelector || ownerLabelSelector != helmSecretsLabelValue {
		log.Info("Cannot use informer to list secrets", "listOptions", opts)
		return w.inner.List(ctx, opts)
	}

	secrets, err := w.informerLister.List(labelSelector)
	if err != nil {
		return nil, err
	}

	secretList := &corev1.SecretList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items:    make([]corev1.Secret, len(secrets)),
	}
	for i, sec := range secrets {
		secretList.Items[i] = *sec
	}

	return secretList, nil
}

func hasListOptionsOtherThanLabelSelector(opts metav1.ListOptions) bool {
	empty := metav1.ListOptions{}

	providedWithoutLabelSelector := opts
	providedWithoutLabelSelector.LabelSelector = ""

	return empty != providedWithoutLabelSelector
}

func (w *WatchedSecrets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return w.inner.Watch(ctx, opts)
}

func (w *WatchedSecrets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, _ ...string) (result *corev1.Secret, err error) {
	return w.inner.Patch(ctx, name, pt, data, opts)
}

func (w *WatchedSecrets) Apply(ctx context.Context, secret *applyconfv1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Secret, err error) {
	return w.inner.Apply(ctx, secret, opts)
}

var _ typedcorev1.SecretInterface = &WatchedSecrets{}

func NewWatchedSecrets(clientSet kubernetes.Interface, namespace string) *WatchedSecrets {
	log.V(2).Info("Get secrets client", "namespace", namespace)

	helmListOptionsTweaker := func(options *metav1.ListOptions) {
		labelSelector, err := labels.Parse(options.LabelSelector)
		if err != nil {
			log.Info("Could not parse labelSelector", "labelSelector", options.LabelSelector)
			panic("could not parse labelSelector")
		}

		ownerLabelSelector, hasOwnerLabelSelector := labelSelector.RequiresExactMatch(helmSecretsLabelKey)

		if !hasOwnerLabelSelector || ownerLabelSelector != helmSecretsLabelValue {
			helmRequirement, _ := labels.NewRequirement(
				"owner", selection.Equals, []string{helmSecretsLabelValue},
			)
			labelSelectorWithOwner := labelSelector.Add(*helmRequirement)
			options.LabelSelector = labelSelectorWithOwner.String()
		}
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(clientSet, time.Second*30, informers.WithNamespace(namespace), informers.WithTweakListOptions(helmListOptionsTweaker))
	secretsInformer := informerFactory.Core().V1().Secrets()

	informerSecretsLister := secretsInformer.Lister().Secrets(namespace)

	return &WatchedSecrets{
		inner:           clientSet.CoreV1().Secrets(namespace),
		informerFactory: informerFactory,
		informerLister:  informerSecretsLister,
	}
}

func (w *WatchedSecrets) Run() {
	w.informerFactory.Start(wait.NeverStop)
	_ = w.informerFactory.WaitForCacheSync(wait.NeverStop)
}
