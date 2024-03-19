package ecoSystem

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

type DoguInterface interface {
	Create(ctx context.Context, dogu *v1.Dogu, opts metav1.CreateOptions) (*v1.Dogu, error)
	Update(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (*v1.Dogu, error)
	UpdateSpecWithRetry(ctx context.Context, dogu *v1.Dogu, modifySpecFn func(spec v1.DoguSpec) v1.DoguSpec, opts metav1.UpdateOptions) (result *v1.Dogu, err error)
	UpdateStatus(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (*v1.Dogu, error)
	UpdateStatusWithRetry(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1.UpdateOptions) (result *v1.Dogu, err error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Dogu, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.DoguList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Dogu, err error)
}

type doguClient struct {
	client rest.Interface
	ns     string
}

// Get takes name of the dogu, and returns the corresponding dogu object, and an error if there is any.
func (d *doguClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Dogu, err error) {
	result = &v1.Dogu{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("dogus").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Dogus that match those selectors.
func (d *doguClient) List(ctx context.Context, opts metav1.ListOptions) (result *v1.DoguList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.DoguList{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("dogus").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested dogus.
func (d *doguClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return d.client.Get().
		Namespace(d.ns).
		Resource("dogus").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a dogu and creates it.  Returns the server's representation of the dogu, and an error, if there is any.
func (d *doguClient) Create(ctx context.Context, dogu *v1.Dogu, opts metav1.CreateOptions) (result *v1.Dogu, err error) {
	result = &v1.Dogu{}
	err = d.client.Post().
		Namespace(d.ns).
		Resource("dogus").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a dogu and updates it. Returns the server's representation of the dogu, and an error, if there is any.
func (d *doguClient) Update(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (result *v1.Dogu, err error) {
	result = &v1.Dogu{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("dogus").
		Name(dogu.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// UpdateSpecWithRetry updates the spec of the resource, retrying if a conflict error arises.
func (d *doguClient) UpdateSpecWithRetry(ctx context.Context, dogu *v1.Dogu, modifySpecFn func(spec v1.DoguSpec) v1.DoguSpec, opts metav1.UpdateOptions) (result *v1.Dogu, err error) {
	firstTry := true

	var currentObj *v1.Dogu
	err = retry.OnConflict(func() error {
		if firstTry {
			firstTry = false
			currentObj = dogu.DeepCopy()
		} else {
			currentObj, err = d.Get(ctx, dogu.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		currentObj.Spec = modifySpecFn(currentObj.Spec)
		currentObj, err = d.Update(ctx, currentObj, opts)
		return err
	})
	if err != nil {
		return nil, err
	}

	return currentObj, nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *doguClient) UpdateStatus(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (result *v1.Dogu, err error) {
	result = &v1.Dogu{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("dogus").
		Name(dogu.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(dogu).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatusWithRetry updates the status of the resource, retrying if a conflict error arises.
func (d *doguClient) UpdateStatusWithRetry(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1.UpdateOptions) (result *v1.Dogu, err error) {
	firstTry := true

	var currentObj *v1.Dogu
	err = retry.OnConflict(func() error {
		if firstTry {
			firstTry = false
			currentObj = dogu.DeepCopy()
		} else {
			currentObj, err = d.Get(ctx, dogu.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		currentObj.Status = modifyStatusFn(currentObj.Status)
		currentObj, err = d.UpdateStatus(ctx, currentObj, opts)
		return err
	})
	if err != nil {
		return nil, err
	}

	return currentObj, nil
}

// Delete takes name of the dogu and deletes it. Returns an error if one occurs.
func (d *doguClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.client.Delete().
		Namespace(d.ns).
		Resource("dogus").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (d *doguClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return d.client.Delete().
		Namespace(d.ns).
		Resource("dogus").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched dogu.
func (d *doguClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Dogu, err error) {
	result = &v1.Dogu{}
	err = d.client.Patch(pt).
		Namespace(d.ns).
		Resource("dogus").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
