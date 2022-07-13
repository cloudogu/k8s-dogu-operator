package controllers

import (
	"context"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

func Test_evaluateRequiredOperation(t *testing.T) {
	testDoguCr := &k8sv1.Dogu{}
	logger := log.FromContext(context.TODO())

	t.Run("installed should return upgrade", func(t *testing.T) {
		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled}

		operation, err := evaluateRequiredOperation(testDoguCr, logger)

		require.NoError(t, err)
		assert.Equal(t, Upgrade, operation)
	})

	t.Run("deletiontimestamp should return delete", func(t *testing.T) {
		now := v1.NewTime(time.Now())
		testDoguCr.DeletionTimestamp = &now

		operation, err := evaluateRequiredOperation(testDoguCr, logger)

		require.NoError(t, err)
		assert.Equal(t, Delete, operation)
		testDoguCr.DeletionTimestamp = nil
	})

	t.Run("installing should return ignore", func(t *testing.T) {
		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalling}

		operation, err := evaluateRequiredOperation(testDoguCr, logger)

		require.NoError(t, err)
		assert.Equal(t, Ignore, operation)
	})

	t.Run("deleting should return ignore", func(t *testing.T) {
		testDoguCr.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting}

		operation, err := evaluateRequiredOperation(testDoguCr, logger)

		require.NoError(t, err)
		assert.Equal(t, Ignore, operation)
	})

	t.Run("default should return ignore", func(t *testing.T) {
		testDoguCr.Status = k8sv1.DoguStatus{Status: "youaresomethingelse"}

		operation, err := evaluateRequiredOperation(testDoguCr, logger)

		require.NoError(t, err)
		assert.Equal(t, Ignore, operation)
	})
}
