package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_upgradeabilityChecker_Check(t *testing.T) {
	t.Run("should succeed without forceUpgrade", func(t *testing.T) {
		// given
		upgradeVersion := "4.2.3-11"

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = upgradeVersion
		sut := NewUpgradeabilityChecker()

		// when
		err := sut.Check(fromDogu, toDogu, false)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail for downgrade without forceUpgrade", func(t *testing.T) {
		// given
		upgradeVersion := "1.2.3-4"

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = upgradeVersion
		sut := NewUpgradeabilityChecker()

		// when
		err := sut.Check(fromDogu, toDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "upgradeability check failed: remote version must be greater than local version '1.2.3-4 > 4.2.3-10'", err.Error())
	})
	t.Run("should succeed for downgrade with forceUpgrade", func(t *testing.T) {
		// given
		upgradeVersion := "1.2.3-4"

		fromDogu := readTestDataDogu(t, redmineBytes)
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = upgradeVersion
		sut := NewUpgradeabilityChecker()

		// when
		err := sut.Check(fromDogu, toDogu, true)

		// then
		require.NoError(t, err)
	})
}
