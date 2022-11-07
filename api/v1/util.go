package v1

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

func OnErrorRetry(maxTries int, retriable func(error) bool, workload func() error) error {
	err := retry.OnError(wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    maxTries,
		Cap:      3 * time.Minute,
	}, retriable, workload)

	if retriable(err) {
		return fmt.Errorf("the maximum number of retries was reached: %w", err)
	}
	return err
}

func OnErrorRetryAlways(maxTries int, fn func() error) error {
	return OnErrorRetry(maxTries, func(err error) bool {
		return true
	}, fn)
}
