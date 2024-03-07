package ecoSystem

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"time"
)

type DoguRestartInterface interface {
	Create(ctx context.Context, dogu *v1.DoguRestart, opts metav1.CreateOptions) (*v1.DoguRestart, error)
	Update(ctx context.Context, dogu *v1.DoguRestart, opts metav1.UpdateOptions) (*v1.DoguRestart, error)
	UpdateStatus(ctx context.Context, dogu *v1.DoguRestart, opts metav1.UpdateOptions) (*v1.DoguRestart, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.DoguRestart, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.DoguRestartList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.DoguRestart, err error)
}

type doguRestartClient struct {
	client rest.Interface
	ns     string
}

// Get takes name of the dogu restart, and returns the corresponding dogu restart object, and an error if there is any.
func (d *doguRestartClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.DoguRestart, err error) {
	result = &v1.DoguRestart{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("dogurestarts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of dogu restarts that match those selectors.
func (d *doguRestartClient) List(ctx context.Context, opts metav1.ListOptions) (result *v1.DoguRestartList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.DoguRestartList{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("dogurestarts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested dogu restarts.
func (d *doguRestartClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return d.client.Get().
		Namespace(d.ns).
		Resource("dogurestarts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a dogu restart and creates it.  Returns the server's representation of the dogu restart, and an error, if there is any.
func (d *doguRestartClient) Create(ctx context.Context, dogu *v1.DoguRestart, opts metav1.CreateOptions) (result *v1.DoguRestart, err error) {
	result = &v1.DoguRestart{}
	err = d.client.Post().
		Namespace(d.ns).
		Resource("dogurestarts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a dogu restart and updates it. Returns the server's representation of the dogu restart, and an error, if there is any.
func (d *doguRestartClient) Update(ctx context.Context, dogu *v1.DoguRestart, opts metav1.UpdateOptions) (result *v1.DoguRestart, err error) {
	result = &v1.DoguRestart{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("dogurestarts").
		Name(dogu.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *doguRestartClient) UpdateStatus(ctx context.Context, dogu *v1.DoguRestart, opts metav1.UpdateOptions) (result *v1.DoguRestart, err error) {
	result = &v1.DoguRestart{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("dogurestarts").
		Name(dogu.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the dogu restart and deletes it. Returns an error if one occurs.
func (d *doguRestartClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.client.Delete().
		Namespace(d.ns).
		Resource("dogurestarts").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (d *doguRestartClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return d.client.Delete().
		Namespace(d.ns).
		Resource("dogurestarts").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched dogu restart.
func (d *doguRestartClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.DoguRestart, err error) {
	result = &v1.DoguRestart{}
	err = d.client.Patch(pt).
		Namespace(d.ns).
		Resource("dogurestarts").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
