/*
Copyright 2018 The play-operator Authors

Commercial software license.
*/package v1alpha1

import (
	v1alpha1 "github.com/coreos/play/pkg/apis/play/v1alpha1"
	scheme "github.com/coreos/play/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PlayServicesGetter has a method to return a PlayServiceInterface.
// A group's client should implement this interface.
type PlayServicesGetter interface {
	PlayServices(namespace string) PlayServiceInterface
}

// PlayServiceInterface has methods to work with PlayService resources.
type PlayServiceInterface interface {
	Create(*v1alpha1.PlayService) (*v1alpha1.PlayService, error)
	Update(*v1alpha1.PlayService) (*v1alpha1.PlayService, error)
	UpdateStatus(*v1alpha1.PlayService) (*v1alpha1.PlayService, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.PlayService, error)
	List(opts v1.ListOptions) (*v1alpha1.PlayServiceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PlayService, err error)
	PlayServiceExpansion
}

// playServices implements PlayServiceInterface
type playServices struct {
	client rest.Interface
	ns     string
}

// newPlayServices returns a PlayServices
func newPlayServices(c *PlayV1alpha1Client, namespace string) *playServices {
	return &playServices{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the playService, and returns the corresponding playService object, and an error if there is any.
func (c *playServices) Get(name string, options v1.GetOptions) (result *v1alpha1.PlayService, err error) {
	result = &v1alpha1.PlayService{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("playservices").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PlayServices that match those selectors.
func (c *playServices) List(opts v1.ListOptions) (result *v1alpha1.PlayServiceList, err error) {
	result = &v1alpha1.PlayServiceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("playservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested playServices.
func (c *playServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("playservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a playService and creates it.  Returns the server's representation of the playService, and an error, if there is any.
func (c *playServices) Create(playService *v1alpha1.PlayService) (result *v1alpha1.PlayService, err error) {
	result = &v1alpha1.PlayService{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("playservices").
		Body(playService).
		Do().
		Into(result)
	return
}

// Update takes the representation of a playService and updates it. Returns the server's representation of the playService, and an error, if there is any.
func (c *playServices) Update(playService *v1alpha1.PlayService) (result *v1alpha1.PlayService, err error) {
	result = &v1alpha1.PlayService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("playservices").
		Name(playService.Name).
		Body(playService).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *playServices) UpdateStatus(playService *v1alpha1.PlayService) (result *v1alpha1.PlayService, err error) {
	result = &v1alpha1.PlayService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("playservices").
		Name(playService.Name).
		SubResource("status").
		Body(playService).
		Do().
		Into(result)
	return
}

// Delete takes name of the playService and deletes it. Returns an error if one occurs.
func (c *playServices) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("playservices").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *playServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("playservices").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched playService.
func (c *playServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PlayService, err error) {
	result = &v1alpha1.PlayService{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("playservices").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
