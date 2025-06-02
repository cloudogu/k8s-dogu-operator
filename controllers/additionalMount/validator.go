package additionalMount

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/retry-lib/retry"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type requeueableValidationError struct {
	wrapped error
}

func (r *requeueableValidationError) Unwrap() error {
	return r.wrapped
}

func (r *requeueableValidationError) Error() string {
	return r.wrapped.Error()
}

func (r *requeueableValidationError) Requeue() bool {
	return true
}

var sourceWaitLimit = time.Minute

type Validator struct {
	configMapInterface configMapGetter
	secretInterface    secretGetter
}

func NewValidator(configMapGetter configMapGetter, secretGatter secretGetter) *Validator {
	return &Validator{
		configMapInterface: configMapGetter,
		secretInterface:    secretGatter,
	}
}

// ValidateAdditionalMounts validates the additional mounts from the dogu resource and dogu.json
func (v *Validator) ValidateAdditionalMounts(ctx context.Context, doguDescriptor *core.Dogu, doguResource *k8sv2.Dogu) error {
	var multiErr []error
	var additionalMounts = make(map[k8sv2.DataMount]struct{})

	if len(doguResource.Spec.AdditionalMounts) > 0 && !hasVolumeWithName(doguDescriptor, "localConfig") {
		multiErr = append(multiErr, fmt.Errorf("dogu %s has no local config volume needed by addtional data mounts", doguResource.Name))
	}

	for _, dataMount := range doguResource.Spec.AdditionalMounts {
		// check for duplicate entries
		if _, ok := additionalMounts[dataMount]; ok {
			multiErr = append(multiErr, fmt.Errorf("duplicate entry %+v", dataMount))
			continue
		}
		additionalMounts[dataMount] = struct{}{}

		// check for valid dogu descriptor volume references
		if !hasVolumeWithName(doguDescriptor, dataMount.Volume) {
			multiErr = append(multiErr, fmt.Errorf("volume %s does not exists in dogu descriptor for dogu %s", dataMount.Volume, doguResource.Name))
		}

		// check if the source really exists
		err := v.validateSource(ctx, dataMount)
		if err != nil {
			multiErr = append(multiErr, &requeueableValidationError{err})
		}
	}

	return errors.Join(multiErr...)
}

func hasVolumeWithName(dogu *core.Dogu, volume string) bool {
	for _, doguVolume := range dogu.Volumes {
		if doguVolume.Name == volume {
			return true
		}
	}
	return false
}

func (v *Validator) validateSource(ctx context.Context, mount k8sv2.DataMount) error {
	switch mount.SourceType {
	case k8sv2.DataSourceConfigMap:
		return v.checkIfResourceExists(ctx, v.configMapResourceGetter, mount.Name)
	case k8sv2.DataSourceSecret:
		return v.checkIfResourceExists(ctx, v.secretResourceGetter, mount.Name)
	default:
		return fmt.Errorf("unknown additional mount type %s for dogu", mount.SourceType)
	}
}

func (v *Validator) checkIfResourceExists(ctx context.Context, resourceChecker func(ctx context.Context, name string) error, name string) error {
	return retry.OnErrorWithLimit(sourceWaitLimit, doNotRetryOnNotFoundOrNil, func() error {
		return resourceChecker(ctx, name)
	})
}

func (v *Validator) configMapResourceGetter(ctx context.Context, name string) error {
	_, err := v.configMapInterface.Get(ctx, name, v1.GetOptions{})
	return err
}

func (v *Validator) secretResourceGetter(ctx context.Context, name string) error {
	_, err := v.secretInterface.Get(ctx, name, v1.GetOptions{})
	return err
}

var doNotRetryOnNotFoundOrNil = func(err error) bool {
	if err == nil || apierrors.IsNotFound(err) {
		return false
	}

	return true
}
