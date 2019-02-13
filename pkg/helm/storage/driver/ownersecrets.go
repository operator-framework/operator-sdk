// Copyright 2019 The Operator-SDK Authors
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

package driver

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	rspb "k8s.io/helm/pkg/proto/hapi/release"
	helmdriver "k8s.io/helm/pkg/storage/driver"
	storageerrors "k8s.io/helm/pkg/storage/errors"
)

var _ helmdriver.Driver = (*OwnerSecrets)(nil)

// OwnerSecretsDriverName is the string name of the driver.
const OwnerSecretsDriverName = "OwnerSecret"

// OwnerSecrets is a wrapper around an implementation of a kubernetes
// SecretsInterface. It is intended to be used when a release is owned
// by an in-cluster object.
type OwnerSecrets struct {
	impl     corev1.SecretInterface
	ownerRef metav1.OwnerReference
	Log      func(string, ...interface{})
}

// NewOwnerSecrets initializes a new OwnerSecrets wrapping an implementation of
// the kubernetes SecretsInterface. The provided ownerRef is used in three ways:
//
// First, it is included as an owner reference on each secret created by the
// returned storage driver
//
// Second, it is used to construct the name of release secrets so that they are
// isolated from releases managed by other storage drivers that share the same
// release name.
//
// Third, it is used to override the "OWNER" label that is used in queries and
// included created secrets.
func NewOwnerSecrets(ownerRef metav1.OwnerReference, impl corev1.SecretInterface) *OwnerSecrets {
	return &OwnerSecrets{
		impl:     impl,
		ownerRef: ownerRef,
		Log:      func(_ string, _ ...interface{}) {},
	}
}

// Name returns the name of the driver.
func (secrets *OwnerSecrets) Name() string {
	return OwnerSecretsDriverName
}

// Get fetches the release named by key. The corresponding release is returned
// or error if not found.
func (secrets *OwnerSecrets) Get(key string) (*rspb.Release, error) {
	secretName := fmt.Sprintf("%s-%s", shortenUID(secrets.ownerRef.UID), key)

	// fetch the secret holding the release named by secretName
	obj, err := secrets.impl.Get(secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, storageerrors.ErrReleaseNotFound(key)
		}

		secrets.Log("get: failed to get %q: %s", key, err)
		return nil, err
	}
	// found the secret, decode the base64 data string
	r, err := decodeRelease(string(obj.Data["release"]))
	if err != nil {
		secrets.Log("get: failed to decode data %q: %s", key, err)
		return nil, err
	}
	// return the release object
	return r, nil
}

// List fetches all releases and returns the list releases such
// that filter(release) == true. An error is returned if the
// secret fails to retrieve the releases.
func (secrets *OwnerSecrets) List(filter func(*rspb.Release) bool) ([]*rspb.Release, error) {
	owner := string(secrets.ownerRef.UID)
	lsel := kblabels.Set{"OWNER": owner}.AsSelector()
	opts := metav1.ListOptions{LabelSelector: lsel.String()}

	list, err := secrets.impl.List(opts)
	if err != nil {
		secrets.Log("list: failed to list: %s", err)
		return nil, err
	}

	var results []*rspb.Release

	// iterate over the secrets object list
	// and decode each release
	for _, item := range list.Items {
		rls, err := decodeRelease(string(item.Data["release"]))
		if err != nil {
			secrets.Log("list: failed to decode release: %v: %s", item, err)
			continue
		}
		if filter(rls) {
			results = append(results, rls)
		}
	}
	return results, nil
}

// Query fetches all releases that match the provided map of labels.
// An error is returned if the secret fails to retrieve the releases.
func (secrets *OwnerSecrets) Query(labels map[string]string) ([]*rspb.Release, error) {
	// Set or override the OWNER label since we know what it should be.
	labels["OWNER"] = string(secrets.ownerRef.UID)

	ls := kblabels.Set{}
	for k, v := range labels {
		if errs := validation.IsValidLabelValue(v); len(errs) != 0 {
			return nil, fmt.Errorf("invalid label value: %q: %s", v, strings.Join(errs, "; "))
		}
		ls[k] = v
	}

	opts := metav1.ListOptions{LabelSelector: ls.AsSelector().String()}

	list, err := secrets.impl.List(opts)
	if err != nil {
		secrets.Log("query: failed to query with labels: %s", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, storageerrors.ErrReleaseNotFound(labels["NAME"])
	}

	var results []*rspb.Release
	for _, item := range list.Items {
		rls, err := decodeRelease(string(item.Data["release"]))
		if err != nil {
			secrets.Log("query: failed to decode release: %s", err)
			continue
		}
		results = append(results, rls)
	}
	return results, nil
}

// Create creates a new Secret holding the release. If the
// Secret already exists, ErrReleaseExists is returned.
func (secrets *OwnerSecrets) Create(key string, rls *rspb.Release) error {
	// set labels for secrets object meta data
	var lbs labels

	lbs.init()
	lbs.set("CREATED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new secret to hold the release
	obj, err := newOwnerSecretsObject(secrets.ownerRef, key, rls, lbs)
	if err != nil {
		secrets.Log("create: failed to encode release %q: %s", rls.Name, err)
		return err
	}
	// push the secret object out into the kubiverse
	if _, err := secrets.impl.Create(obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return storageerrors.ErrReleaseExists(key)
		}

		secrets.Log("create: failed to create: %s", err)
		return err
	}
	return nil
}

// Update updates the Secret holding the release. If not found
// the Secret is created to hold the release.
func (secrets *OwnerSecrets) Update(key string, rls *rspb.Release) error {
	// set labels for secrets object meta data
	var lbs labels

	lbs.init()
	lbs.set("MODIFIED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new secret object to hold the release
	obj, err := newOwnerSecretsObject(secrets.ownerRef, key, rls, lbs)
	if err != nil {
		secrets.Log("update: failed to encode release %q: %s", rls.Name, err)
		return err
	}
	// push the secret object out into the kubiverse
	_, err = secrets.impl.Update(obj)
	if err != nil {
		secrets.Log("update: failed to update: %s", err)
		return err
	}
	return nil
}

// Delete deletes the Secret holding the release named by key.
func (secrets *OwnerSecrets) Delete(key string) (rls *rspb.Release, err error) {
	secretName := fmt.Sprintf("%s-%s", shortenUID(secrets.ownerRef.UID), key)

	// fetch the release to check existence
	if rls, err = secrets.Get(key); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, storageerrors.ErrReleaseExists(rls.Name)
		}

		secrets.Log("delete: failed to get release %q: %s", key, err)
		return nil, err
	}
	// delete the release
	if err = secrets.impl.Delete(secretName, &metav1.DeleteOptions{}); err != nil {
		return rls, err
	}
	return rls, nil
}

// newOwnerSecretsObject constructs a kubernetes Secret object
// to store a release. Each secret data entry is the base64
// encoded string of a release's binary protobuf encoding.
//
// The following labels are used within each secret:
//
//    "MODIFIED_AT"    - timestamp indicating when this secret was last modified. (set in Update)
//    "CREATED_AT"     - timestamp indicating when this secret was created. (set in Create)
//    "VERSION"        - version of the release.
//    "STATUS"         - status of the release (see proto/hapi/release.status.pb.go for variants)
//    "OWNER"          - owner of the secret, uid of the owner reference.
//    "NAME"           - name of the release.
//
func newOwnerSecretsObject(ownerRef metav1.OwnerReference, key string, rls *rspb.Release, lbs labels) (*v1.Secret, error) {
	// encode the release
	s, err := encodeRelease(rls)
	if err != nil {
		return nil, err
	}

	if lbs == nil {
		lbs.init()
	}

	// apply labels
	lbs.set("NAME", rls.Name)
	lbs.set("OWNER", string(ownerRef.UID))
	lbs.set("STATUS", rspb.Status_Code_name[int32(rls.Info.Status.Code)])
	lbs.set("VERSION", strconv.Itoa(int(rls.Version)))

	// create and return secret object
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-%s", shortenUID(ownerRef.UID), key),
			Labels:          lbs.toMap(),
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Data: map[string][]byte{"release": []byte(s)},
	}, nil
}
