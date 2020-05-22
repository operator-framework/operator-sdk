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

package olm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	apiextinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/olm"
	internalolmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// TODO(estroz): figure out a good way to deal with creating scorecard objects
// and injecting proxy container

const (
	defaultTimeout   = time.Minute * 2
	defaultNamespace = "default"

	installModeFormat = "InstallModeType[=ns1,ns2[, ...]]"
)

func init() {
	// OLM schemes must be added to the global Scheme so controller-runtime's
	// client recognizes OLM objects.
	apiextinstall.Install(scheme.Scheme)
	if err := operatorsv1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("Failed to add OLM operator API v1 types to scheme: %v", err)
	}
}

// OperatorCmd configures deployment and teardown of an operator via OLM.
// Intended to be used by an exported struct, as it lackas a Run method.
type OperatorCmd struct {
	// KubeconfigPath is the local path to a kubeconfig. This uses well-defined
	// default loading rules to load the config if empty.
	KubeconfigPath string
	// OperatorNamespace is the cluster namespace in which operator resources
	// are created.
	// OperatorNamespace must already exist in the cluster or be defined in
	// a manifest passed to IncludePaths.
	OperatorNamespace string
	// OLMNamespace is the namespace in which OLM is installed.
	OLMNamespace string
	// InstallMode specifies which supported installMode should be used to
	// create an OperatorGroup. The format for this field is as follows:
	//
	// "InstallModeType=[ns1,ns2[, ...]]"
	//
	// The InstallModeType string passed must be marked as "supported" in the
	// CSV being installed. The namespaces passed must exist or be created by
	// passing a Namespace manifest to IncludePaths. An empty set of namespaces
	// can be used for AllNamespaces.
	InstallMode string
	// Timeout dictates how long to wait for a REST call to complete. A call
	// exceeding Timeout will generate an error.
	Timeout time.Duration
	// ForceRegistry forces deletion of registry resources.
	ForceRegistry bool

	once sync.Once
}

func (c *OperatorCmd) AddToFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeconfigPath, "kubeconfig", "",
		"The file path to kubernetes configuration file. Defaults to location "+
			"specified by $KUBECONFIG, or to default file rules if not set")
	fs.StringVar(&c.OLMNamespace, "olm-namespace", olm.DefaultOLMNamespace,
		"The namespace where OLM is installed")
	fs.StringVar(&c.OperatorNamespace, "operator-namespace", "",
		"The namespace where operator resources are created. It must already exist "+
			"in the cluster or be defined in a manifest passed to --include")
	fs.StringVar(&c.InstallMode, "install-mode", "",
		"InstallMode to create OperatorGroup with. Format: "+installModeFormat)
	fs.DurationVar(&c.Timeout, "timeout", defaultTimeout,
		"Time to wait for the command to complete before failing")
}

func (c *OperatorCmd) validate() error {
	if c.InstallMode != "" {
		if _, _, err := parseInstallModeKV(c.InstallMode); err != nil {
			return err
		}
	}
	return nil
}

func (c *OperatorCmd) initialize() {
	c.once.Do(func() {
		if c.Timeout <= 0 {
			c.Timeout = defaultTimeout
		}
	})
}

type operatorManager struct {
	client *internalolmclient.Client
	// olmNamespace is the namespace where olm is installed
	// and operator registry server resources are created
	olmNamespace      string
	operatorNamespace string

	installMode      operatorsv1alpha1.InstallModeType //nolint:structcheck
	targetNamespaces []string                          //nolint:structcheck
	olmObjects       []runtime.Object
}

func (c *OperatorCmd) newManager() (*operatorManager, error) {
	m := &operatorManager{}

	// Namespace in which OLM is deployed.
	if m.olmNamespace = c.OLMNamespace; m.olmNamespace == "" {
		m.olmNamespace = olm.DefaultOLMNamespace
	}

	// Cluster and operator namespace info.
	rc, ns, err := k8sutil.GetKubeconfigAndNamespace(c.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace from kubeconfig %s: %w", c.KubeconfigPath, err)
	}
	if ns == "" {
		ns = defaultNamespace
	}
	if m.operatorNamespace = c.OperatorNamespace; m.operatorNamespace == "" {
		m.operatorNamespace = ns
	}
	if m.client == nil {
		m.client, err = internalolmclient.ClientForConfig(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to create SDK OLM client: %w", err)
		}
	}

	return m, nil
}

// TODO(estroz): check registry health on each "status" subcommand invocation
func (m *operatorManager) status(ctx context.Context, us ...*unstructured.Unstructured) internalolmclient.Status {
	objs := []runtime.Object{}
	for _, u := range us {
		uc := u.DeepCopy()
		uc.SetNamespace(m.operatorNamespace)
		objs = append(objs, uc)
	}
	return m.client.GetObjectsStatus(ctx, objs...)
}

func (m operatorManager) hasCatalogSource() bool {
	return containsKind(m.olmObjects, operatorsv1alpha1.CatalogSourceKind)
}

func (m operatorManager) hasSubscription() bool {
	return containsKind(m.olmObjects, operatorsv1alpha1.SubscriptionKind)
}

func (m operatorManager) hasOperatorGroup() bool {
	return containsKind(m.olmObjects, operatorsv1.OperatorGroupKind)
}

func containsKind(objs []runtime.Object, kind string) bool {
	for _, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			return true
		}
	}
	return false
}

func readObjectsFromFile(path string) (objs []*unstructured.Unstructured, err error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
	for scanner.Scan() {
		b, err := yaml.YAMLToJSON(scanner.Bytes())
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON before decode: %v", err)
		}
		u := &unstructured.Unstructured{}
		if err := u.UnmarshalJSON(b); err != nil {
			return nil, fmt.Errorf("failed to decode object from manifest %s: %w", path, err)
		}
		objs = append(objs, u)
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("failed to scan manifest %s: %w", path, scanner.Err())
	}
	if len(objs) == 0 {
		return nil, fmt.Errorf("no objects found in manifest %s", path)
	}
	return objs, nil
}
