package util

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"strings"
	"time"
)

// GetMapKeysAsString returns the key of a map as a string in form: "key1, key2, key3".
func GetMapKeysAsString(input map[string]string) string {
	output := ""

	for key := range input {
		output = fmt.Sprintf("%s, %s", output, key)
	}

	return strings.TrimLeft(output, ", ")
}

func OnErrorRetry(maxTries int, retriable func(error) bool, fn func() error) error {
	return retry.OnError(wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    maxTries,
		Cap:      3 * time.Minute,
	}, retriable, fn)
}

func OnErrorRetryAlways(maxTries int, fn func() error) error {
	return OnErrorRetry(maxTries, func(err error) bool {
		return true
	}, fn)
}
