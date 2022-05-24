package dependency_test

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type validatorCheckerSuccess struct {
	called bool
}

func (v *validatorCheckerSuccess) ValidateAllDependencies(_ *core.Dogu) error {
	v.called = true
	return nil
}

type validatorCheckerError struct {
	called bool
}

func (v *validatorCheckerError) ValidateAllDependencies(_ *core.Dogu) error {
	v.called = true
	return fmt.Errorf("some error")
}

func TestDependencyChecker_ValidateDependencies(t *testing.T) {
	t.Run("successfully check all deps with multiple Validators", func(t *testing.T) {
		// given
		checkerOne := &validatorCheckerSuccess{}
		checkerTwo := &validatorCheckerSuccess{}
		checkerThree := &validatorCheckerSuccess{}
		compositeValidator := dependency.CompositeDependencyValidator{Validators: []dependency.DependencyValidator{
			checkerOne, checkerTwo, checkerThree,
		}}

		// when
		err := compositeValidator.ValidateDependencies(&core.Dogu{})

		// then
		require.NoError(t, err)
		assert.True(t, checkerOne.called)
		assert.True(t, checkerTwo.called)
		assert.True(t, checkerThree.called)
	})

	t.Run("return error when one Validators returns error", func(t *testing.T) {
		// given
		checkerOne := &validatorCheckerSuccess{}
		checkerTwo := &validatorCheckerError{}
		checkerThree := &validatorCheckerSuccess{}
		compositeValidator := dependency.CompositeDependencyValidator{Validators: []dependency.DependencyValidator{
			checkerOne, checkerTwo, checkerThree,
		}}

		// when
		err := compositeValidator.ValidateDependencies(&core.Dogu{})

		// then
		require.Error(t, err)
		assert.True(t, checkerOne.called)
		assert.True(t, checkerTwo.called)
		assert.True(t, checkerThree.called)
		assert.Contains(t, err.Error(), "some error")
	})
}

func TestNewCompositeDependencyValidator(t *testing.T) {
	t.Run("successfully create new checker", func(t *testing.T) {
		// given
		version, err := core.ParseVersion("0.0.0")
		require.NoError(t, err)

		cesRegistryMock := &cesmocks.DoguRegistry{}

		// when
		compositeValidator := dependency.NewCompositeDependencyValidator(&version, cesRegistryMock)

		// then
		assert.NotNil(t, compositeValidator)
		assert.NotNil(t, compositeValidator.Validators)
		assert.NotNil(t, compositeValidator.Validators[0])
		assert.NotNil(t, compositeValidator.Validators[1])
	})
}
