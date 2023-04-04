package retry

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// AlwaysRetryFunc returns always true and thus indicates that always should be tried until the retrier hits its limit.
var AlwaysRetryFunc = func(err error) bool {
	return true
}

// TestableRetryFunc returns true if the returned error is a TestableRetrierError and indicates that an action should be tried until the retrier hits its limit.
var TestableRetryFunc = func(err error) bool {
	_, ok := err.(*TestableRetrierError)
	return ok
}

// TestableRetrierError marks errors that indicate that a previously executed action should be retried with again. It must wrap an existing error.
type TestableRetrierError struct {
	Err error
}

// Error returns the error's string representation.
func (tre *TestableRetrierError) Error() string {
	return tre.Err.Error()
}

// OnError provides a K8s-way "retrier" mechanism. The value from retriable is used to indicate if workload should
// retried another time. Please see AlwaysRetryFunc() if a workload should always retried until a fixed threshold is
// reached.
func OnError(maxTries int, retriable func(error) bool, workload func() error) error {
	return onError(maxTries, 3*time.Minute, retriable, workload)
}

// OnErrorWithLimit provides a K8s-way "retrier" mechanism with a time limit as option.
func OnErrorWithLimit(limit time.Duration, retriable func(error) bool, workload func() error) error {
	// Use a high integer here to avoid limit the cap with the steps.
	return onError(9999999, limit, retriable, workload)
}

func onError(maxTries int, limit time.Duration, retriable func(error) bool, workload func() error) error {
	err := retry.OnError(wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    maxTries,
		Cap:      limit,
	}, retriable, workload)

	if err != nil && retriable(err) {
		return fmt.Errorf("the maximum number of retries was reached: %w", err)
	}
	return err
}

// OnConflict provides a K8s-way "retrier" mechanism to avoid conflicts on resource updates.
func OnConflict(fn func() error) error {
	return retry.RetryOnConflict(wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    9999,
		Cap:      30 * time.Second,
	}, fn)
}
