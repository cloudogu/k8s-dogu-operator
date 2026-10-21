package garbagecollection

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/restart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DoguRestartGarbageCollector struct {
	doguRestartInterface doguRestartInterface
}

func NewDoguRestartGarbageCollector(doguRestartInterface doguv2.DoguRestartInterface) *DoguRestartGarbageCollector {
	return &DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterface}
}

const (
	restartSuccessfulHistoryLimitEnv         = "DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT"
	restartFailedHistoryLimitEnv             = "DOGU_RESTART_FAILED_HISTORY_LIMIT"
	restartGarbageCollectionDisabledEnv      = "DOGU_RESTART_GARBAGE_COLLECTION_DISABLED"
	fallbackRestartSuccessfulHistoryLimit    = 3
	fallbackRestartFailedHistoryLimit        = 3
	fallbackRestartGarbageCollectionDisabled = false
)

func (r *DoguRestartGarbageCollector) DoGarbageCollection(ctx context.Context, doguName string) error {
	disabled, err := r.garbageCollectionDisabled()
	if err != nil {
		return err
	}
	if disabled {
		return nil
	}

	restarts, err := r.getDoguRestartsForDogu(ctx, doguName)
	if err != nil {
		return err
	}

	// restarts of v2 dogus are tracked by their phase and restarts of v3 dogus by their conditions
	successfulRestarts := filterDoguRestarts(restarts, func(item *k8sv2.DoguRestart) bool {
		return item.Status.Phase == k8sv2.RestartStatusPhaseCompleted || restart.IsSucceeded(item)
	})

	failedRestarts := filterDoguRestarts(restarts, func(item *k8sv2.DoguRestart) bool {
		return item.Status.Phase.IsFailed() || restart.IsFailed(item)
	})

	var errs []error
	errs = append(errs, r.truncateDoguRestartHistory(ctx, successfulRestarts, restartSuccessfulHistoryLimitEnv, fallbackRestartSuccessfulHistoryLimit))
	errs = append(errs, r.truncateDoguRestartHistory(ctx, failedRestarts, restartFailedHistoryLimitEnv, fallbackRestartFailedHistoryLimit))

	return errors.Join(errs...)
}

func filterDoguRestarts(items []k8sv2.DoguRestart, fn func(item *k8sv2.DoguRestart) bool) []k8sv2.DoguRestart {
	var result []k8sv2.DoguRestart
	for _, item := range items {
		if fn(&item) {
			result = append(result, item)
		}
	}

	return result
}

func (r *DoguRestartGarbageCollector) truncateDoguRestartHistory(ctx context.Context, items []k8sv2.DoguRestart, limitEnv string, fallbackHistoryLimit int) error {
	if len(items) == 0 {
		return nil
	}

	historyLimit := fallbackHistoryLimit
	env, found := os.LookupEnv(limitEnv)
	if found {
		atoi, errConvert := strconv.Atoi(env)
		if errConvert != nil {
			return fmt.Errorf("failed to convert history limit %q of dogu restarts: %w", env, errConvert)
		}
		historyLimit = atoi
	}

	if historyLimit < 0 {
		return fmt.Errorf("failed to execute garbage collection because the limit is less than 0: %d", historyLimit)
	}

	amountOfItemsToDelete := len(items) - historyLimit
	if amountOfItemsToDelete <= 0 {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreationTimestamp.Before(&items[j].CreationTimestamp)
	})

	var errs []error
	// We can not delete as collection by .name because the field selector does not support the || operator.
	for i := range amountOfItemsToDelete {
		// the restart may already be deleted by the garbage collection of another reconciliation
		errs = append(errs, client.IgnoreNotFound(r.doguRestartInterface.Delete(ctx, items[i].Name, metav1.DeleteOptions{})))
	}

	return errors.Join(errs...)
}

func (r *DoguRestartGarbageCollector) getDoguRestartsForDogu(ctx context.Context, doguName string) ([]k8sv2.DoguRestart, error) {
	list, err := r.doguRestartInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list dogu restarts for dogu %q: %w", doguName, err)
	}

	var restartsForDogu []k8sv2.DoguRestart
	for _, item := range list.Items {
		if item.Spec.DoguName == doguName {
			restartsForDogu = append(restartsForDogu, item)
		}
	}

	return restartsForDogu, nil
}

func (r *DoguRestartGarbageCollector) garbageCollectionDisabled() (bool, error) {
	garbageCollectionDisabled := fallbackRestartGarbageCollectionDisabled
	env, found := os.LookupEnv(restartGarbageCollectionDisabledEnv)
	if found {
		var errConvert error
		garbageCollectionDisabled, errConvert = strconv.ParseBool(env)
		if errConvert != nil {
			return false, fmt.Errorf("failed to convert garbage collection disabled flag %q of dogu restarts: %w", env, errConvert)
		}
	}

	return garbageCollectionDisabled, nil
}
