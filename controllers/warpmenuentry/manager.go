package warpmenuentry

import (
	"context"
	"fmt"
	"reflect"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	v1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type WarpMenuEntryManager struct {
	client      warpmenuentryClient
	doguFetcher localDoguFetcher
}

func NewWarpMenuEntryManager(client warpMenuEntryV1.WarpMenuEntryInterface,
	doguFetcher cesregistry.LocalDoguFetcher) *WarpMenuEntryManager {
	return &WarpMenuEntryManager{
		client:      client,
		doguFetcher: doguFetcher,
	}
}

func (wm *WarpMenuEntryManager) EnsureWarpMenuEntry(ctx context.Context, doguResource *doguv2.Dogu) error {
	if doguResource == nil {
		return fmt.Errorf("dogu resource must not be nil")
	}
	doguDescriptor, err := wm.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	if !doguShouldBeInWarpMenu(doguDescriptor) {
		logrus.Infof("manoj , The warp menu entry should not be present for this dogu :[%s]", doguResource.Name)
		return wm.deleteWarpMenuEntryIfItExists(ctx, doguResource)
	}

	warpMenuEntry := buildWarpMenuEntry(doguDescriptor, doguResource)

	return wm.ensureWarpMenuEntry(ctx, warpMenuEntry)

}

func (wm *WarpMenuEntryManager) ensureWarpMenuEntry(ctx context.Context, desiredWarpMenuEntry *v1.WarpMenuEntry) error {
	current, err := wm.client.Get(ctx, desiredWarpMenuEntry.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, createErr := wm.client.Create(ctx, desiredWarpMenuEntry, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create warpmenuentry: %w", createErr)
			}
			logf.FromContext(ctx).Info("warp menu entry has been successfully created")
			return nil
		}
	}
	if reflect.DeepEqual(current.Spec, desiredWarpMenuEntry.Spec) && reflect.DeepEqual(current.OwnerReferences, desiredWarpMenuEntry.OwnerReferences) {
		return nil
	}

	current.Spec = desiredWarpMenuEntry.Spec
	current.OwnerReferences = desiredWarpMenuEntry.OwnerReferences

	_, updateErr := wm.client.Update(ctx, current, metav1.UpdateOptions{})
	if updateErr != nil {
		return fmt.Errorf("error updating warp menu entry %w", updateErr)
	} else {
		logf.FromContext(ctx).Info(fmt.Sprintf("warp menu entry has been successfully updated :%s", desiredWarpMenuEntry.Name))
		return nil
	}
}

func (wm *WarpMenuEntryManager) DeleteWarpMenuEntry(ctx context.Context, doguName *cescommons.SimpleName) error {
	//TODO
	return nil
}

func (wm *WarpMenuEntryManager) deleteWarpMenuEntryIfItExists(ctx context.Context, dogu *doguv2.Dogu) error {
	warpMenuEntry, err := wm.client.Get(ctx, dogu.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			//Nothing to do
			return nil
		} else {
			return fmt.Errorf("error updating getting the menu entry %w", err)
		}
	}
	err = wm.client.Delete(ctx, warpMenuEntry.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting the menu entry %w", err)
	}
	return nil
}
