package upgrade

import (
	"errors"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var doguNil *core.Dogu

func Test_doguFetcher_Fetch(t *testing.T) {
	t.Run("should succeed and return both installed dogu and remote upgrade", func(t *testing.T) {
		// given
		toDoguCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		toDoguCr.Spec.Version = upgradeVersion

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = upgradeVersion

		localRegMock := new(localRegMock)
		localRegMock.On("Get", "redmine").Return(fromDogu, nil)
		remoteRegMock := new(remoteRegMock)
		remoteRegMock.On("GetVersion", "official/redmine", upgradeVersion).Return(toDogu, nil)

		sut := NewDoguFetcher(localRegMock, remoteRegMock)

		// when
		localDogu, remoteDogu, err := sut.Fetch(toDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, fromDogu, localDogu)
		assert.Equal(t, toDogu, remoteDogu)
		localRegMock.AssertExpectations(t)
		remoteRegMock.AssertExpectations(t)
	})
	t.Run("should fail because of error in local registry", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion

		localRegMock := new(localRegMock)
		localRegMock.On("Get", "redmine").Return(doguNil, errors.New("localGet"))
		remoteRegMock := new(remoteRegMock)

		sut := NewDoguFetcher(localRegMock, remoteRegMock)

		// when
		_, _, err := sut.Fetch(redmineCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get local dogu descriptor for redmine")
		assert.Contains(t, err.Error(), "localGet")
		localRegMock.AssertExpectations(t)
		remoteRegMock.AssertExpectations(t)
	})
	t.Run("should fail because of error in remote registry", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion

		fromDogu := readTestDataDogu(t, redmineBytes)

		localRegMock := new(localRegMock)
		localRegMock.On("Get", "redmine").Return(fromDogu, nil)
		remoteRegMock := new(remoteRegMock)
		remoteRegMock.On("GetVersion", "official/redmine", upgradeVersion).Return(doguNil, errors.New("remoteGet"))

		sut := NewDoguFetcher(localRegMock, remoteRegMock)

		// when
		_, _, err := sut.Fetch(redmineCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get remote dogu descriptor for official/redmine:4.2.3-11")
		assert.Contains(t, err.Error(), "remoteGet")
		localRegMock.AssertExpectations(t)
		remoteRegMock.AssertExpectations(t)
	})
}

type localRegMock struct {
	mock.Mock
}

func (lr *localRegMock) Get(name string) (*core.Dogu, error) {
	args := lr.Called(name)
	return args.Get(0).(*core.Dogu), args.Error(1)
}

func (lr *localRegMock) Enable(dogu *core.Dogu) error {
	panic("implement me")
}

func (lr *localRegMock) Register(dogu *core.Dogu) error {
	panic("implement me")
}

func (lr *localRegMock) Unregister(name string) error {
	panic("implement me")
}

func (lr *localRegMock) GetAll() ([]*core.Dogu, error) {
	panic("implement me")
}

func (lr *localRegMock) IsEnabled(name string) (bool, error) {
	panic("implement me")
}

type remoteRegMock struct {
	mock.Mock
}

func (rr *remoteRegMock) Get(name string) (*core.Dogu, error) {
	args := rr.Called(name)
	return args.Get(0).(*core.Dogu), args.Error(1)
}

func (rr *remoteRegMock) Create(dogu *core.Dogu) error {
	panic("implement me")
}

func (rr *remoteRegMock) GetVersion(name, version string) (*core.Dogu, error) {
	args := rr.Called(name, version)
	return args.Get(0).(*core.Dogu), args.Error(1)
}

func (rr *remoteRegMock) GetAll() ([]*core.Dogu, error) {
	panic("implement me")
}

func (rr *remoteRegMock) GetVersionsOf(name string) ([]core.Version, error) {
	panic("implement me")
}

func (rr *remoteRegMock) SetUseCache(useCache bool) {
	panic("implement me")
}

func (rr *remoteRegMock) Delete(dogu *core.Dogu) error {
	panic("implement me")
}
