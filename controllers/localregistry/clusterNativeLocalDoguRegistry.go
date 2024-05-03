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
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

const currentVersionKey = "current"

const (
	appLabelKey      = "app"
	appLabelValueCes = "ces"

	doguNameLabelKey = "dogu.name"

	typeLabelKey                    = "k8s.cloudogu.com/type"
	typeLabelValueLocalDoguRegistry = "local-dogu-registry"
)

type clusterNativeLocalDoguRegistry struct {
	configMapClient corev1client.ConfigMapInterface
}

func getSpecConfigMapName(simpleDoguName string) string {
	return fmt.Sprintf("dogu-spec-%s", simpleDoguName)
}

// Enable makes the dogu spec reachable
// by setting the specLocation field in the dogu resources' status.
func (cmr *clusterNativeLocalDoguRegistry) Enable(ctx context.Context, dogu *core.Dogu) error {
	return retry.OnConflict(func() error {
		specConfigMap, err := cmr.getSpecConfigMapForDogu(ctx, dogu.GetSimpleName())
		if err != nil {
			return err
		}

		specConfigMap.Data[currentVersionKey] = dogu.Version
		_, err = cmr.configMapClient.Update(ctx, specConfigMap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update local registry for dogu %q with new version: %w", dogu.GetSimpleName(), err)
		}

		return nil
	})
}

// Register adds the given dogu spec to the local registry.
//
// Adds the dogu spec to the underlying ConfigMap. Creates the ConfigMap if it does not exist.
func (cmr *clusterNativeLocalDoguRegistry) Register(ctx context.Context, dogu *core.Dogu) error {
	doguJson, jsonErr := json.Marshal(dogu)
	if jsonErr != nil {
		jsonErr = fmt.Errorf("failed to serialize dogu.json of %q: %w", dogu.Name, jsonErr)
	}

	specConfigMapName := getSpecConfigMapName(dogu.GetSimpleName())
	return retry.OnConflict(func() error {
		specConfigMap, getErr := cmr.getSpecConfigMapForDogu(ctx, dogu.GetSimpleName())
		if jsonErr != nil || client.IgnoreNotFound(getErr) != nil {
			return errors.Join(jsonErr, getErr)
		}

		if k8sErrs.IsNotFound(getErr) {
			specConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: specConfigMapName,
					Labels: map[string]string{
						appLabelKey:      appLabelValueCes,
						doguNameLabelKey: dogu.GetSimpleName(),
						typeLabelKey:     typeLabelValueLocalDoguRegistry,
					},
				},
				Data: map[string]string{dogu.Version: string(doguJson)},
			}

			_, createErr := cmr.configMapClient.Create(ctx, specConfigMap, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create local registry for dogu %q: %w", dogu.GetSimpleName(), createErr)
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
func (cmr *clusterNativeLocalDoguRegistry) UnregisterAllVersions(ctx context.Context, simpleDoguName string) error {
	err := cmr.configMapClient.Delete(ctx, getSpecConfigMapName(simpleDoguName), metav1.DeleteOptions{})
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete local registry for dogu %q: %w", simpleDoguName, err)
	}

	return nil
}

// GetCurrent retrieves the spec of the referenced dogu's currently installed version
// through the ConfigMap referenced in the specLocation field of the dogu resource's status.
func (cmr *clusterNativeLocalDoguRegistry) GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error) {
	specConfigMap, err := cmr.getSpecConfigMapForDogu(ctx, simpleDoguName)
	if err != nil {
		return nil, err
	}

	return getCurrentFromSpecConfigMap(specConfigMap, simpleDoguName)
}

func getCurrentFromSpecConfigMap(specConfigMap *corev1.ConfigMap, simpleDoguName string) (*core.Dogu, error) {
	currentVersion, exists := specConfigMap.Data[currentVersionKey]
	if !exists {
		return nil, fmt.Errorf("local dogu registry does not contain currently installed version for dogu %q", simpleDoguName)
	}

	doguJson, exists := specConfigMap.Data[currentVersion]
	if !exists {
		return nil, fmt.Errorf("local dogu registry does not contain dogu.json for currently installed version of dogu %q", simpleDoguName)
	}

	var doguSpec *core.Dogu
	err := json.Unmarshal([]byte(doguJson), &doguSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current dogu.json of %q: %w", simpleDoguName, err)
	}

	return doguSpec, nil
}

// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions
// through the ConfigMaps referenced in the specLocation field of the dogu resources' status.
func (cmr *clusterNativeLocalDoguRegistry) GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error) {
	allLocalDoguRegistriesSelector := fmt.Sprintf("%s=%s,%s,%s=%s", appLabelKey, appLabelValueCes, doguNameLabelKey, typeLabelKey, typeLabelValueLocalDoguRegistry)
	registryList, err := cmr.configMapClient.List(ctx, metav1.ListOptions{LabelSelector: allLocalDoguRegistriesSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster native local dogu registries: %w", err)
	}

	var errs []error
	doguSpecs := make([]*core.Dogu, 0, len(registryList.Items))
	for _, localRegistry := range registryList.Items {
		doguSpec, err := getCurrentFromSpecConfigMap(&localRegistry, localRegistry.Labels[doguNameLabelKey])
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
func (cmr *clusterNativeLocalDoguRegistry) IsEnabled(ctx context.Context, simpleDoguName string) (bool, error) {
	specConfigMap, err := cmr.getSpecConfigMapForDogu(ctx, simpleDoguName)
	if err != nil {
		return false, err
	}

	_, enabled := specConfigMap.Data[currentVersionKey]
	return enabled, nil
}

func (cmr *clusterNativeLocalDoguRegistry) getSpecConfigMapForDogu(ctx context.Context, simpleDoguName string) (*corev1.ConfigMap, error) {
	specConfigMapName := getSpecConfigMapName(simpleDoguName)
	specConfigMap, err := cmr.configMapClient.Get(ctx, specConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get local registry for dogu %q: %w", simpleDoguName, err)
	}

	return specConfigMap, nil
}
