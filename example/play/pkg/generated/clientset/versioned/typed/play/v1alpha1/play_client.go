/*
Copyright 2018 The play-operator Authors

Commercial software license.
*/package v1alpha1

import (
	v1alpha1 "github.com/coreos/play/pkg/apis/play/v1alpha1"
	"github.com/coreos/play/pkg/generated/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type PlayV1alpha1Interface interface {
	RESTClient() rest.Interface
	PlayServicesGetter
}

// PlayV1alpha1Client is used to interact with features provided by the play.example.com group.
type PlayV1alpha1Client struct {
	restClient rest.Interface
}

func (c *PlayV1alpha1Client) PlayServices(namespace string) PlayServiceInterface {
	return newPlayServices(c, namespace)
}

// NewForConfig creates a new PlayV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*PlayV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PlayV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new PlayV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *PlayV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PlayV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *PlayV1alpha1Client {
	return &PlayV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *PlayV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
