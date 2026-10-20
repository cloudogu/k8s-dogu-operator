// Package restart restarts the workloads of a v3 dogu by setting a timestamp annotation in their pod templates.
package restart

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AnnotationKey is set in spec.template.metadata.annotations of all workloads of a dogu to restart them.
	AnnotationKey = "k8s.cloudogu.com/restartAt"
	// FieldManager owns the restart annotation, which makes the operator visible as its owner in managedFields.
	// The annotation survives Flux drift detection and Helm upgrades because it is not part of the chart manifest.
	FieldManager = "k8s-dogu-operator-restart"
	// TimeFormat is the format used to parse and format timestamps.
	TimeFormat = time.RFC3339
)

// workloadDiscovery finds the live workloads of a dogu.
type workloadDiscovery interface {
	Workloads(ctx context.Context, namespace, doguName string) (workloads.Workloads, error)
}

// Restarter restarts the workloads of a dogu.
type Restarter struct {
	k8s       client.Client
	discovery workloadDiscovery
}

// NewRestarter creates a new Restarter.
func NewRestarter(k8s client.Client) *Restarter {
	return &Restarter{
		k8s:       k8s,
		discovery: workloads.NewDiscovery(k8s),
	}
}

// applyConfigFunc builds the server-side apply payload. Apply configurations are used instead of typed objects,
// because they only serialize the fields that are set. A typed object would also send zero values (e.g. selector,
// strategy, containers), so the restart field manager would claim fields that are owned by Helm.
type applyConfigFunc func(annotations map[string]string) runtime.ApplyConfiguration

// workload is the part of a Deployment, StatefulSet, or DaemonSet that the restart needs.
type workload struct {
	kind        string
	name        string
	annotations map[string]string
	ready       bool
	applyConfig applyConfigFunc
}

// Apply sets the restart annotation with restartAt on all workloads of the dogu.
// A newer restart is never overwritten, so applying an older restart later on does not restart the dogu again.
// Workloads that already carry the current restartAt are not changed.
// It returns true if the restart is superseded, i.e. all workloads already carry a newer restart.
func (r *Restarter) Apply(ctx context.Context, namespace, doguName string, restartAt time.Time) (superseded bool, err error) {
	foundWorkloads, err := r.getWorkloads(ctx, namespace, doguName)
	if err != nil {
		return false, fmt.Errorf("failed to get workloads for dogu %s in namespace %s: %w", doguName, namespace, err)
	}

	restartAt = truncateTimestamp(restartAt)
	value := format(restartAt)
	superseded = len(foundWorkloads) > 0

	var errs []error
	for _, w := range foundWorkloads {
		current, ok := parse(w.annotations)
		if ok && current.After(restartAt) {
			continue
		}

		superseded = false
		if ok && current.Equal(restartAt) {
			continue
		}

		applyConfig := w.applyConfig(map[string]string{AnnotationKey: value})
		if aErr := r.k8s.Apply(ctx, applyConfig, client.FieldOwner(FieldManager), client.ForceOwnership); aErr != nil {
			errs = append(errs, fmt.Errorf("failed to set restart annotation on %s/%s of dogu %q: %w", w.kind, w.name, doguName, aErr))
		}
	}

	return superseded, errors.Join(errs...)
}

// RolledOut returns true if the dogu has workloads and all of them carry restartAt or a newer restart and are ready.
// Otherwise, it returns a message that lists the workloads that are not rolled out yet.
func (r *Restarter) RolledOut(ctx context.Context, namespace, doguName string, restartAt time.Time) (bool, string, error) {
	foundWorkloads, err := r.getWorkloads(ctx, namespace, doguName)
	if err != nil {
		return false, "", fmt.Errorf("failed to get workloads for dogu %q: %w", doguName, err)
	}

	if len(foundWorkloads) == 0 {
		return false, "the dogu has no workloads", nil
	}

	restartAt = truncateTimestamp(restartAt)
	var pending []string
	for _, w := range foundWorkloads {
		// a workload without the restart is not rolled out even if it is ready, e.g. if it was read from a stale cache
		current, ok := parse(w.annotations)
		if !ok || current.Before(restartAt) || !w.ready {
			pending = append(pending, fmt.Sprintf("%s/%s", w.kind, w.name))
		}
	}

	if len(pending) > 0 {
		return false, fmt.Sprintf("workloads not rolled out: %s", strings.Join(pending, ", ")), nil
	}

	return true, "", nil
}

func (r *Restarter) getWorkloads(ctx context.Context, namespace, doguName string) ([]workload, error) {
	found, err := r.discovery.Workloads(ctx, namespace, doguName)
	if err != nil {
		return nil, fmt.Errorf("failed to get workloads of dogu %q: %w", doguName, err)
	}

	var result []workload
	for _, d := range found.Deployments {
		result = append(result, workload{
			kind:        "Deployment",
			name:        d.Name,
			annotations: d.Spec.Template.Annotations,
			ready:       workloads.IsDeploymentReady(d),
			applyConfig: func(annotations map[string]string) runtime.ApplyConfiguration {
				return appsv1.Deployment(d.Name, d.Namespace).
					WithSpec(appsv1.DeploymentSpec().
						WithTemplate(podTemplate(annotations)))
			},
		})
	}

	for _, s := range found.StatefulSets {
		result = append(result, workload{
			kind:        "StatefulSet",
			name:        s.Name,
			annotations: s.Spec.Template.Annotations,
			ready:       workloads.IsStatefulSetReady(s),
			applyConfig: func(annotations map[string]string) runtime.ApplyConfiguration {
				return appsv1.StatefulSet(s.Name, s.Namespace).
					WithSpec(appsv1.StatefulSetSpec().
						WithTemplate(podTemplate(annotations)))
			},
		})
	}

	for _, ds := range found.DaemonSets {
		result = append(result, workload{
			kind:        "DaemonSet",
			name:        ds.Name,
			annotations: ds.Spec.Template.Annotations,
			ready:       workloads.IsDaemonSetReady(ds),
			applyConfig: func(annotations map[string]string) runtime.ApplyConfiguration {
				return appsv1.DaemonSet(ds.Name, ds.Namespace).
					WithSpec(appsv1.DaemonSetSpec().
						WithTemplate(podTemplate(annotations)))
			},
		})
	}

	return result, nil
}

func podTemplate(annotations map[string]string) *corev1ac.PodTemplateSpecApplyConfiguration {
	return corev1ac.PodTemplateSpec().WithAnnotations(annotations)
}

func format(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}

// parse returns the restart timestamp of the annotations. An unparsable timestamp counts as missing.
func parse(annotations map[string]string) (time.Time, bool) {
	value, ok := annotations[AnnotationKey]
	if !ok {
		return time.Time{}, false
	}

	t, err := time.Parse(TimeFormat, value)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

func truncateTimestamp(t time.Time) time.Time {
	return t.Truncate(time.Second)
}
