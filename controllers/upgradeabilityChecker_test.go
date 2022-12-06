package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_upgradeabilityChecker_Check(t *testing.T) {
	t.Run("should succeed without forceUpgrade", func(t *testing.T) {
		// given
		upgradeVersion := "4.2.3-11"

		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu.Version = upgradeVersion
		sut := &upgradeChecker{}

		// when
		actual, err := sut.IsUpgradeable(fromDogu, toDogu, false)

		// then
		require.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("should fail for downgrade without forceUpgrade", func(t *testing.T) {
		// given
		actuallyDowngradeVersion := "1.2.3-4"

		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu.Version = actuallyDowngradeVersion
		sut := &upgradeChecker{}

		// when
		_, err := sut.IsUpgradeable(fromDogu, toDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "downgrade from 4.2.3-10 to 1.2.3-4 is not allowed", err.Error())
	})
	t.Run("should succeed but return false for equal versions", func(t *testing.T) {
		// given
		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		sut := &upgradeChecker{}

		// when
		actual, err := sut.IsUpgradeable(fromDogu, toDogu, false)

		// then
		require.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("should succeed for downgrade with forceUpgrade", func(t *testing.T) {
		// given
		upgradeVersion := "1.2.3-4"

		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu.Version = upgradeVersion
		sut := &upgradeChecker{}

		// when
		actual, err := sut.IsUpgradeable(fromDogu, toDogu, true)

		// then
		require.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("should fails for unparsable fromDogu Version", func(t *testing.T) {
		// given
		upgradeVersion := "1.2.3-4"

		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		fromDogu.Version = "ö#a.b.c.-.-.d.e.-..f.g--¹"
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu.Version = upgradeVersion
		sut := &upgradeChecker{}

		// when
		_, err := sut.IsUpgradeable(fromDogu, toDogu, false)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not check upgradeability of local dogu")
	})
	t.Run("should fails for unparsable toDogu Version", func(t *testing.T) {
		// given
		fromDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		toDogu.Version = "ö#a.b.c.-.-.d.e.-..f.g--¹"
		sut := &upgradeChecker{}

		// when
		_, err := sut.IsUpgradeable(fromDogu, toDogu, false)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not check upgradeability of remote dogu")
	})
}
