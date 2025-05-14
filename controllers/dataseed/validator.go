package dataseed

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

// ValidateDataSeeds validates the data seed mounts
func (v *Validator) ValidateDataSeeds(ctx context.Context, doguDescriptor *core.Dogu, doguResource *k8sv2.Dogu) error {
	var multiErr []error
	var dataMounts = make(map[k8sv2.DataMount]struct{})
	for _, dataMount := range doguResource.Spec.Data {
		// check for duplicate entries
		if _, ok := dataMounts[dataMount]; ok {
			multiErr = append(multiErr, fmt.Errorf("duplicate entry %+v", dataMount))
			continue
		}
		dataMounts[dataMount] = struct{}{}

		// check for valid dogu descriptor volume references
		volumeFound := false
		// volumeClientFound := false
		for _, doguVolume := range doguDescriptor.Volumes {
			if doguVolume.Name == dataMount.Volume {
				// TODO check volume clients?
				// if len(doguVolume.Clients) > 0 {
				// 	volumeClientFound = true
				// }
				volumeFound = true
				break
			}
		}
		if !volumeFound {
			multiErr = append(multiErr, fmt.Errorf("volume %s does not exists in dogu descriptor for dogu %s", dataMount.Volume, doguResource.Name))
		}

		// check if the source really exists
		multiErr = append(multiErr, v.validateSource(ctx, dataMount))
	}

	return errors.Join(multiErr...)
}

func (v *Validator) validateSource(ctx context.Context, mount k8sv2.DataMount) error {
	switch mount.SourceType {
	case k8sv2.DataSourceConfigMap:
		return v.checkIfResourceExists(ctx, v.configMapResourceGetter, mount.Name)
	case k8sv2.DataSourceSecret:
		return v.checkIfResourceExists(ctx, v.secretResourceGetter, mount.Name)
	default:
		return fmt.Errorf("unknown data mount type %s for dogu", mount.SourceType)
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
