package upgrade

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/mock"

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
	t.Run("should fail for different dogu namespaces", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = remoteDogu.GetNamespace() + "/test"
		// when
		err := checkDoguIdentity(localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogus must have the same name (ldap=test)", err.Error())
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
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

		mockedChecker := new(premiseMock)
		mockedChecker.On("CheckWithResource", fromDoguResource).Return(nil)
		mockedChecker.On("ValidateDependencies", fromDogu).Return(nil)
		mockedChecker.On("CheckDependenciesRecursive", fromDogu, fromDoguResource.Namespace).Return(nil)

		sut := NewPremisesChecker(mockedChecker, mockedChecker, mockedChecker)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.NoError(t, err)
		mockedChecker.AssertExpectations(t)
	})
	t.Run("should fail when dogu identity check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Name = "somethingdifferent"

		mockedChecker := new(premiseMock)
		mockedChecker.On("CheckWithResource", fromDoguResource).Return(nil)
		mockedChecker.On("ValidateDependencies", fromDogu).Return(nil)
		mockedChecker.On("CheckDependenciesRecursive", fromDogu, fromDoguResource.Namespace).Return(nil)

		sut := NewPremisesChecker(mockedChecker, mockedChecker, mockedChecker)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dogus must have the same name")
		// there is no assert.IsNoType() assertion so we test it by negative type assertion
		_, ok := err.(*requeueablePremisesError)
		assert.False(t, ok)
		mockedChecker.AssertExpectations(t)
	})
	t.Run("should fail when dependency validator fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedChecker := new(premiseMock)
		mockedChecker.On("CheckWithResource", fromDoguResource).Return(errors.New("CheckWithResource"))

		sut := NewPremisesChecker(mockedChecker, mockedChecker, mockedChecker)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CheckWithResource")
		assert.IsType(t, &requeueablePremisesError{}, err)
		// prove that the above negative type assertion works by positive type assertion
		_, ok := err.(*requeueablePremisesError)
		assert.True(t, ok)
		mockedChecker.AssertExpectations(t)
	})
	t.Run("should fail when dogu health check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedChecker := new(premiseMock)
		mockedChecker.On("CheckWithResource", fromDoguResource).Return(nil)
		mockedChecker.On("ValidateDependencies", fromDogu).Return(errors.New("ValidateDependencies"))

		sut := NewPremisesChecker(mockedChecker, mockedChecker, mockedChecker)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ValidateDependencies")
		assert.IsType(t, &requeueablePremisesError{}, err)
		mockedChecker.AssertExpectations(t)
	})
	t.Run("should fail when dogu dependency health check fails", func(t *testing.T) {
		fromDoguResource := readTestDataRedmineCr(t)
		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)

		mockedChecker := new(premiseMock)
		mockedChecker.On("CheckWithResource", fromDoguResource).Return(nil)
		mockedChecker.On("ValidateDependencies", fromDogu).Return(nil)
		mockedChecker.On("CheckDependenciesRecursive", fromDogu, fromDoguResource.Namespace).Return(errors.New("CheckDependenciesRecursive"))

		sut := NewPremisesChecker(mockedChecker, mockedChecker, mockedChecker)

		// when
		err := sut.Check(ctx, fromDoguResource, fromDogu, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CheckDependenciesRecursive")
		assert.IsType(t, &requeueablePremisesError{}, err)
		mockedChecker.AssertExpectations(t)
	})
}

type premiseMock struct {
	mock.Mock
}

func (pm *premiseMock) ValidateDependencies(_ context.Context, dogu *core.Dogu) error {
	args := pm.Called(dogu)
	return args.Error(0)
}

func (pm *premiseMock) CheckDependenciesRecursive(_ context.Context, fromDogu *core.Dogu, currentK8sNamespace string) error {
	args := pm.Called(fromDogu, currentK8sNamespace)
	return args.Error(0)
}

func (pm *premiseMock) CheckWithResource(_ context.Context, doguResource *k8sv1.Dogu) error {
	args := pm.Called(doguResource)
	return args.Error(0)
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
