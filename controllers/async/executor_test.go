package async

import (
	"context"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type testStep struct{}

func (ts *testStep) GetStartCondition() string {
	return "start"
}

func (ts *testStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	return "finished", nil
}

type errorStep struct{}

func (es *errorStep) GetStartCondition() string {
	return "errorStart"
}

func (es *errorStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	return "end", assert.AnError
}

func TestNewAsyncExecutionController(t *testing.T) {
	result := NewDoguExecutionController()

	require.NotNil(t, result)
}

func Test_asyncExecutionController_AddStep(t *testing.T) {
	// given
	executor := &doguExecutionController{}
	step := &testStep{}

	// when
	executor.AddStep(step)

	// then
	assert.Equal(t, 1, len(executor.steps))
}

func Test_asyncExecutionController_Execute(t *testing.T) {
	t.Run("should do nothing and return nil if state is finished", func(t *testing.T) {
		// given
		executor := &doguExecutionController{}

		// when
		err := executor.Execute(context.TODO(), nil, "finished")

		// then
		require.Nil(t, err)
	})

	t.Run("should return error if steps are empty", func(t *testing.T) {
		// given
		executor := &doguExecutionController{}

		// when
		err := executor.Execute(context.TODO(), nil, "start")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to find state in step list")
	})

	t.Run("should return error if condition has no corresponding step", func(t *testing.T) {
		// given
		executor := &doguExecutionController{}
		step := &testStep{}
		executor.AddStep(step)

		// when
		err := executor.Execute(context.TODO(), nil, "mid")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to find state in step list")
	})

	t.Run("should return error if a step returns an error", func(t *testing.T) {
		// given
		executor := &doguExecutionController{}
		step := &errorStep{}
		executor.AddStep(step)

		// when
		err := executor.Execute(context.TODO(), nil, "errorStart")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, assert.AnError, err)
	})

	t.Run("should execute the corresponding step and call the next step", func(t *testing.T) {
		// given
		executor := &doguExecutionController{}
		step0 := &errorStep{}
		step1 := &testStep{}
		executor.AddStep(step0)
		executor.AddStep(step1)

		// when
		err := executor.Execute(context.TODO(), nil, "start")

		// then
		require.Nil(t, err)
	})
}
