package localregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

type ClusterNativeLocalDoguRegistry struct {
	doguClient      ecoSystem.DoguInterface
	configMapClient corev1client.ConfigMapInterface
}

func getConfigMapName(dogu *core.Dogu) string {
	return fmt.Sprintf("dogu-spec-%s-%s", dogu.GetNamespace(), dogu.GetSimpleName())
}

// Enable makes the dogu spec reachable
// by setting the specLocation field in the dogu resources' status.
func (cmr *ClusterNativeLocalDoguRegistry) Enable(ctx context.Context, dogu *core.Dogu) error {
	return retry.OnConflict(func() error {
		doguResource, err := cmr.doguClient.Get(ctx, dogu.GetSimpleName(), metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get dogu cr %q: %w", dogu.Name, err)
		}

		doguResource.Status.SpecLocation = getConfigMapName(dogu)
		_, err = cmr.doguClient.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update spec location in status of dogu cr %q: %w", dogu.Name, err)
		}

		return nil
	})
}

// Register adds the given dogu spec to the local registry.
//
// Adds the dogu spec to the underlying ConfigMap. Creates the ConfigMap if it does not exist.
func (cmr *ClusterNativeLocalDoguRegistry) Register(ctx context.Context, dogu *core.Dogu) error {
	doguJson, jsonErr := json.Marshal(dogu)
	if jsonErr != nil {
		jsonErr = fmt.Errorf("failed to serialize dogu.json of %q: %w", dogu.Name, jsonErr)
	}

	configMapName := getConfigMapName(dogu)
	return retry.OnConflict(func() error {
		specConfigMap, getErr := cmr.configMapClient.Get(ctx, configMapName, metav1.GetOptions{})
		if client.IgnoreNotFound(getErr) != nil {
			getErr = fmt.Errorf("failed to get local registry for dogu %q: %w", dogu.Name, getErr)
		}

		if jsonErr != nil || client.IgnoreNotFound(getErr) != nil {
			return errors.Join(jsonErr, getErr)
		}

		if k8sErrs.IsNotFound(getErr) {
			specConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: configMapName,
					Labels: map[string]string{
						"app":       "ces",
						"dogu.name": dogu.GetSimpleName(),
					},
				},
				Data: map[string]string{dogu.Version: string(doguJson)},
			}

			_, createErr := cmr.configMapClient.Create(ctx, specConfigMap, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create local registry entry for dogu %q: %w", dogu.Name, createErr)
			}

			return nil
		}

		specConfigMap.Data[dogu.Version] = string(doguJson)
		_, updateErr := cmr.configMapClient.Update(ctx, specConfigMap, metav1.UpdateOptions{})
		if updateErr != nil {
			return fmt.Errorf("failed to add local registry entry for dogu %q: %w", dogu.Name, updateErr)
		}

		return nil
	})
}

// UnregisterAllVersions deletes all versions of the dogu spec from the local registry and makes the spec unreachable.
//
// Deletes the backing ConfigMap. Resetting the specLocation field in the dogu resource's status is not necessary
// as the resource will either be deleted or the field will be overwritten.
func (cmr *ClusterNativeLocalDoguRegistry) UnregisterAllVersions(ctx context.Context, doguName string) error {
	doguResource, err := cmr.doguClient.Get(ctx, doguName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get CR for dogu %q: %w", doguName, err)
	}

	err = cmr.configMapClient.Delete(ctx, doguResource.Status.SpecLocation, metav1.DeleteOptions{})
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete local registry for dogu %q: %w", doguName, err)
	}

	return nil
}

// Reregister adds the new dogu spec to the local registry, enables it, and deletes all specs referenced by the old dogu name.
func (cmr *ClusterNativeLocalDoguRegistry) Reregister(ctx context.Context, dogu *core.Dogu) error {
	err := cmr.UnregisterAllVersions(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to unregister old versions of dogu %q: %w", dogu.GetSimpleName(), err)
	}

	err = cmr.Register(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to reregister new version of dogu %q: %w", dogu.Name, err)
	}

	err = cmr.Enable(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to enable new version of dogu %q: %w", dogu.Name, err)
	}

	return nil
}

// GetCurrent retrieves the spec of the referenced dogu's currently installed version
// through the ConfigMap referenced in the specLocation field of the dogu resource's status.
func (cmr *ClusterNativeLocalDoguRegistry) GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error) {
	doguResource, err := cmr.doguClient.Get(ctx, simpleDoguName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CR for dogu %q: %w", simpleDoguName, err)
	}

	return cmr.getCurrentByDoguResource(ctx, doguResource)
}

func (cmr *ClusterNativeLocalDoguRegistry) getCurrentByDoguResource(ctx context.Context, doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	specConfigMap, err := cmr.configMapClient.Get(ctx, doguResource.Status.SpecLocation, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get local registry ConfigMap for dogu %q: %w", doguResource.Spec.Name, err)
	}

	doguJson := specConfigMap.Data[doguResource.Status.InstalledVersion]

	var doguSpec *core.Dogu
	err = json.Unmarshal([]byte(doguJson), doguSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current dogu.json of %q: %w", doguResource.Spec.Name, err)
	}

	return doguSpec, nil
}

// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions
// through the ConfigMaps referenced in the specLocation field of the dogu resources' status.
func (cmr *ClusterNativeLocalDoguRegistry) GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error) {
	doguList, err := cmr.doguClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list dogu CRs: %w", err)
	}

	var errs []error
	doguSpecs := make([]*core.Dogu, 0, len(doguList.Items))
	for _, doguResource := range doguList.Items {
		doguSpec, err := cmr.getCurrentByDoguResource(ctx, &doguResource)
		errs = append(errs, err)
		doguSpecs = append(doguSpecs, doguSpec)
	}

	err = errors.Join(errs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get some dogu specs: %w", err)
	}

	return doguSpecs, nil
}

// IsEnabled checks if the current spec of the referenced dogu is reachable
// by verifying that the specLocation field in the dogu resource's status is set.
func (cmr *ClusterNativeLocalDoguRegistry) IsEnabled(ctx context.Context, simpleDoguName string) (bool, error) {
	doguResource, err := cmr.doguClient.Get(ctx, simpleDoguName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get CR for dogu %q: %w", simpleDoguName, err)
	}

	return doguResource.Status.SpecLocation != "", nil
}
