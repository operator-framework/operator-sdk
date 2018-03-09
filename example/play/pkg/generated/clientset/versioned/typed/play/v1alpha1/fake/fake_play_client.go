/*
Copyright 2018 The play-operator Authors

Commercial software license.
*/package fake

import (
	v1alpha1 "github.com/coreos/play/pkg/generated/clientset/versioned/typed/play/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakePlayV1alpha1 struct {
	*testing.Fake
}

func (c *FakePlayV1alpha1) PlayServices(namespace string) v1alpha1.PlayServiceInterface {
	return &FakePlayServices{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakePlayV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
