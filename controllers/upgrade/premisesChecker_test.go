package upgrade

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_checkDoguIdentity(t *testing.T) {
	t.Run("should succeed for dogus when forceUpgrade is off and remote dogu has a higher version", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Version = "2.4.48-5"

		// when
		err := checkDoguIdentity(localDogu, remoteDogu, false)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed for dogus when forceUpgrade is on but would originally fail because of versions or names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/ldap"

		// when
		err := checkDoguIdentity(localDogu, remoteDogu, true)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = remoteDogu.GetNamespace() + "/test"
		// when
		err := checkDoguIdentity(localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogus must have the same name (ldap=test)", err.Error())
	})
	t.Run("should fail for different dogu namespaces", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/" + remoteDogu.GetSimpleName()
		// when
		err := checkDoguIdentity(localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogus must have the same namespace (official=different-ns)", err.Error())
	})
}

func Test_premisesChecker_Check(t *testing.T) {
	ctx := context.TODO()

	t.Run("should succeed", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedDependencyValidator.On("ValidateDependencies", ctx, fromDogu).Return(nil)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(nil)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)
		mockedRecursiveHealthChecker.On("CheckDependenciesRecursive", ctx, fromDogu, "").Return(nil)
		mockedSecurityValidator := newMockSecurityValidator(t)
		mockedSecurityValidator.EXPECT().ValidateSecurity(toDogu, fromDoguResource).Return(nil)
		mockedDoguDataSeedValidator := newMockDoguDataSeedValidator(t)
		mockedDoguDataSeedValidator.EXPECT().ValidateDataSeeds(testCtx, toDogu, fromDoguResource).Return(nil)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, mockedSecurityValidator, mockedDoguDataSeedValidator)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail when security validation fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedDependencyValidator.On("ValidateDependencies", ctx, fromDogu).Return(nil)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(nil)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)
		mockedRecursiveHealthChecker.On("CheckDependenciesRecursive", ctx, fromDogu, "").Return(nil)
		mockedSecurityValidator := newMockSecurityValidator(t)
		mockedSecurityValidator.EXPECT().ValidateSecurity(toDogu, fromDoguResource).Return(assert.AnError)
		mockedDoguDataSeedValidator := newMockDoguDataSeedValidator(t)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, mockedSecurityValidator, mockedDoguDataSeedValidator)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail when dogu data seed validation fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedDependencyValidator.On("ValidateDependencies", ctx, fromDogu).Return(nil)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(nil)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)
		mockedRecursiveHealthChecker.On("CheckDependenciesRecursive", ctx, fromDogu, "").Return(nil)
		mockedSecurityValidator := newMockSecurityValidator(t)
		mockedSecurityValidator.EXPECT().ValidateSecurity(toDogu, fromDoguResource).Return(nil)
		mockedDoguDataSeedValidator := newMockDoguDataSeedValidator(t)
		mockedDoguDataSeedValidator.EXPECT().ValidateDataSeeds(testCtx, toDogu, fromDoguResource).Return(assert.AnError)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, mockedSecurityValidator, mockedDoguDataSeedValidator)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail when dogu identity check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Name = "somethingdifferent"

		sut := NewPremisesChecker(nil, nil, nil, nil, nil)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogus must have the same name")
		// there is no assert.IsNoType() assertion so we test it by negative type assertion
		_, ok := err.(*requeueablePremisesError)
		assert.False(t, ok)
	})
	t.Run("should fail when dependency validator fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(assert.AnError)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, nil, nil)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.IsType(t, &requeueablePremisesError{}, err)
		// prove that the above negative type assertion works by positive type assertion
		_, ok := err.(*requeueablePremisesError)
		assert.True(t, ok)
	})
	t.Run("should fail when dogu health check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedDependencyValidator.On("ValidateDependencies", ctx, fromDogu).Return(assert.AnError)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(nil)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, nil, nil)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.IsType(t, &requeueablePremisesError{}, err)
	})
	t.Run("should fail when dogu dependency health check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedDependencyValidator := NewMockDependencyValidator(t)
		mockedDependencyValidator.On("ValidateDependencies", ctx, fromDogu).Return(nil)
		mockedHealthChecker := newMockDoguHealthChecker(t)
		mockedHealthChecker.On("CheckByName", ctx, fromDoguResource.GetObjectKey()).Return(nil)
		mockedRecursiveHealthChecker := newMockDoguRecursiveHealthChecker(t)
		mockedRecursiveHealthChecker.On("CheckDependenciesRecursive", ctx, fromDogu, "").Return(assert.AnError)

		sut := NewPremisesChecker(mockedDependencyValidator, mockedHealthChecker, mockedRecursiveHealthChecker, nil, nil)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.IsType(t, &requeueablePremisesError{}, err)
	})
}

func Test_requeueablePremisesError(t *testing.T) {
	assert.Error(t, &requeueablePremisesError{})
}

func Test_requeueablePremisesError_Error(t *testing.T) {
	sut := &requeueablePremisesError{assert.AnError}
	assert.Equal(t, assert.AnError.Error(), sut.Error())
}

func Test_requeueablePremisesError_Requeue(t *testing.T) {
	sut := &requeueablePremisesError{assert.AnError}
	assert.True(t, sut.Requeue())
}

func Test_requeueablePremisesError_Unwrap(t *testing.T) {
	sut := &requeueablePremisesError{assert.AnError}

	actual := sut.Unwrap()

	assert.Same(t, assert.AnError, actual)
	expectedWrap := fmt.Errorf("%w", assert.AnError)
	actualWrap := fmt.Errorf("%w", sut)
	assert.NotSame(t, expectedWrap, actualWrap)
	assert.Equal(t, expectedWrap.Error(), actualWrap.Error())
}
