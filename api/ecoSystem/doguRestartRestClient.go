package ecoSystem

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/retry-lib/retry"
)

type DoguRestartInterface interface {
	Create(ctx context.Context, dogu *v2.DoguRestart, opts metav1.CreateOptions) (*v2.DoguRestart, error)
	Update(ctx context.Context, dogu *v2.DoguRestart, opts metav1.UpdateOptions) (*v2.DoguRestart, error)
	UpdateSpecWithRetry(ctx context.Context, doguRestart *v2.DoguRestart, modifySpecFn func(spec v2.DoguRestartSpec) v2.DoguRestartSpec, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error)
	UpdateStatus(ctx context.Context, dogu *v2.DoguRestart, opts metav1.UpdateOptions) (*v2.DoguRestart, error)
	UpdateStatusWithRetry(ctx context.Context, doguRestart *v2.DoguRestart, modifyStatusFn func(v2.DoguRestartStatus) v2.DoguRestartStatus, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v2.DoguRestart, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v2.DoguRestartList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v2.DoguRestart, err error)
}

type doguRestartClient struct {
	client rest.Interface
	ns     string
}

// Get takes name of the dogu restart, and returns the corresponding dogu restart object, and an error if there is any.
func (d *doguRestartClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v2.DoguRestart, err error) {
	result = &v2.DoguRestart{}
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
func (d *doguRestartClient) List(ctx context.Context, opts metav1.ListOptions) (result *v2.DoguRestartList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v2.DoguRestartList{}
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
func (d *doguRestartClient) Create(ctx context.Context, dogu *v2.DoguRestart, opts metav1.CreateOptions) (result *v2.DoguRestart, err error) {
	result = &v2.DoguRestart{}
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
func (d *doguRestartClient) Update(ctx context.Context, dogu *v2.DoguRestart, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error) {
	result = &v2.DoguRestart{}
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

// UpdateSpecWithRetry updates the spec of the resource, retrying if a conflict error arises.
func (d *doguRestartClient) UpdateSpecWithRetry(ctx context.Context, doguRestart *v2.DoguRestart, modifySpecFn func(spec v2.DoguRestartSpec) v2.DoguRestartSpec, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error) {
	firstTry := true

	var currentObj *v2.DoguRestart
	err = retry.OnConflict(func() error {
		if firstTry {
			firstTry = false
			currentObj = doguRestart.DeepCopy()
		} else {
			currentObj, err = d.Get(ctx, doguRestart.Name, metav1.GetOptions{})
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

// UpdateStatus updates the status of the resource.
func (d *doguRestartClient) UpdateStatus(ctx context.Context, dogu *v2.DoguRestart, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error) {
	result = &v2.DoguRestart{}
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

// UpdateStatusWithRetry updates the status of the resource, retrying if a conflict error arises.
func (d *doguRestartClient) UpdateStatusWithRetry(ctx context.Context, doguRestart *v2.DoguRestart, modifyStatusFn func(v2.DoguRestartStatus) v2.DoguRestartStatus, opts metav1.UpdateOptions) (result *v2.DoguRestart, err error) {
	firstTry := true

	var currentObj *v2.DoguRestart
	err = retry.OnConflict(func() error {
		if firstTry {
			firstTry = false
			currentObj = doguRestart.DeepCopy()
		} else {
			currentObj, err = d.Get(ctx, doguRestart.Name, metav1.GetOptions{})
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
func (d *doguRestartClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v2.DoguRestart, err error) {
	result = &v2.DoguRestart{}
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
