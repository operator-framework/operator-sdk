// Copyright 2020 The Operator-SDK Authors
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
	"context"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Configuration struct {
	Namespace      string
	ServiceAccount string
	KubeconfigPath string
	RESTConfig     *rest.Config
	Client         client.Client
	Scheme         *runtime.Scheme
	Timeout        time.Duration

	overrides *clientcmd.ConfigOverrides
}

func (c *Configuration) BindFlags(fs *pflag.FlagSet) {
	if c.overrides == nil {
		c.overrides = &clientcmd.ConfigOverrides{}
	}
	clientcmd.BindOverrideFlags(c.overrides, fs, clientcmd.ConfigOverrideFlags{
		ContextOverrideFlags: clientcmd.ContextOverrideFlags{
			Namespace: clientcmd.FlagInfo{
				LongName:    "namespace",
				ShortName:   "n",
				Default:     "",
				Description: "If present, namespace scope for this CLI request",
			},
		},
	})
	fs.StringVar(&c.KubeconfigPath, "kubeconfig", "",
		"Path to the kubeconfig file to use for CLI requests.")
	fs.StringVar(&c.ServiceAccount, "service-account", "",
		"Service account name to bind registry objects to. If unset, the default service account is used. "+
			"This value does not override the operator's service account")
	fs.DurationVar(&c.Timeout, "timeout", 2*time.Minute,
		"Duration to wait for the command to complete before failing")
}

func (c *Configuration) Load() error {
	if c.overrides == nil {
		c.overrides = &clientcmd.ConfigOverrides{}
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = c.KubeconfigPath
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		return err
	}
	cfg := clientcmd.NewDefaultClientConfig(*mergedConfig, c.overrides)
	cc, err := cfg.ClientConfig()
	if err != nil {
		return err
	}

	ns, _, err := cfg.Namespace()
	if err != nil {
		return err
	}

	sch := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		v1alpha1.AddToScheme,
		v1.AddToScheme,
		apiextv1.AddToScheme,
	} {
		if err := f(sch); err != nil {
			return err
		}
	}
	cl, err := client.New(cc, client.Options{
		Scheme: sch,
	})
	if err != nil {
		return err
	}

	c.Scheme = sch
	c.Client = &operatorClient{cl}
	if c.Namespace == "" {
		c.Namespace = ns
	}
	c.RESTConfig = cc

	return nil
}

type operatorClient struct {
	client.Client
}

func (c *operatorClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	opts = append(opts, client.FieldOwner("operator-sdk"))
	return c.Client.Create(ctx, obj, opts...)
}
