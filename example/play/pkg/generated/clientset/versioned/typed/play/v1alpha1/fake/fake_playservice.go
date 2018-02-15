/*
Copyright 2018 The play-operator Authors

Commercial software license.
*/package fake

import (
	v1alpha1 "github.com/coreos/play/pkg/apis/play/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePlayServices implements PlayServiceInterface
type FakePlayServices struct {
	Fake *FakePlayV1alpha1
	ns   string
}

var playservicesResource = schema.GroupVersionResource{Group: "play.example.com", Version: "v1alpha1", Resource: "playservices"}

var playservicesKind = schema.GroupVersionKind{Group: "play.example.com", Version: "v1alpha1", Kind: "PlayService"}

// Get takes name of the playService, and returns the corresponding playService object, and an error if there is any.
func (c *FakePlayServices) Get(name string, options v1.GetOptions) (result *v1alpha1.PlayService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(playservicesResource, c.ns, name), &v1alpha1.PlayService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PlayService), err
}

// List takes label and field selectors, and returns the list of PlayServices that match those selectors.
func (c *FakePlayServices) List(opts v1.ListOptions) (result *v1alpha1.PlayServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(playservicesResource, playservicesKind, c.ns, opts), &v1alpha1.PlayServiceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PlayServiceList{}
	for _, item := range obj.(*v1alpha1.PlayServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested playServices.
func (c *FakePlayServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(playservicesResource, c.ns, opts))

}

// Create takes the representation of a playService and creates it.  Returns the server's representation of the playService, and an error, if there is any.
func (c *FakePlayServices) Create(playService *v1alpha1.PlayService) (result *v1alpha1.PlayService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(playservicesResource, c.ns, playService), &v1alpha1.PlayService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PlayService), err
}

// Update takes the representation of a playService and updates it. Returns the server's representation of the playService, and an error, if there is any.
func (c *FakePlayServices) Update(playService *v1alpha1.PlayService) (result *v1alpha1.PlayService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(playservicesResource, c.ns, playService), &v1alpha1.PlayService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PlayService), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePlayServices) UpdateStatus(playService *v1alpha1.PlayService) (*v1alpha1.PlayService, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(playservicesResource, "status", c.ns, playService), &v1alpha1.PlayService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PlayService), err
}

// Delete takes name of the playService and deletes it. Returns an error if one occurs.
func (c *FakePlayServices) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(playservicesResource, c.ns, name), &v1alpha1.PlayService{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePlayServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(playservicesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.PlayServiceList{})
	return err
}

// Patch applies the patch and returns the patched playService.
func (c *FakePlayServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PlayService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(playservicesResource, c.ns, name, data, subresources...), &v1alpha1.PlayService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PlayService), err
}
