package upgrade

import (
	"testing"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/stretchr/testify/assert"
)

// adrDescriptor is the upgrade descriptor from the example of ADR-0059.
var adrDescriptor = []doguv3.Upgrade{
	{From: ">=1.0.0 <=1.7.0", To: "1.8.0"},
	{From: ">=1.8.0-0 <2.0.0", To: "2.0.0", IsMigration: true},
}

func TestValidatePath(t *testing.T) {
	t.Run("upgrade without descriptor is valid", func(t *testing.T) {
		err := ValidatePath("1.0.0", "2.0.0", nil)

		assert.NoError(t, err)
	})

	t.Run("installed version outside of the descriptor has no upgrade path", func(t *testing.T) {
		err := ValidatePath("0.9.0", "2.0.0", adrDescriptor)

		assert.ErrorIs(t, err, ErrNoUpgradePath)
	})

	t.Run("pre-release outside of a range without pre-releases has no upgrade path", func(t *testing.T) {
		err := ValidatePath("1.5.0-3", "2.0.0", adrDescriptor)

		assert.ErrorIs(t, err, ErrNoUpgradePath)
	})

	t.Run("target version outside of the descriptor has no upgrade path", func(t *testing.T) {
		err := ValidatePath("1.5.0", "2.1.0", adrDescriptor)

		assert.ErrorIs(t, err, ErrNoUpgradePath)
	})

	validTests := []struct {
		name      string
		installed string
		target    string
	}{
		{name: "within the same range", installed: "1.2.0", target: "1.5.0"},
		{name: "to the version of the matching entry", installed: "1.5.0", target: "1.8.0"},
		{name: "across several entries", installed: "1.5.0", target: "2.0.0"},
		{name: "into the range of a following entry", installed: "1.5.0", target: "1.9.0"},
		{name: "from a pre-release within a range that includes pre-releases", installed: "1.8.0-rc.1", target: "2.0.0"},
		{name: "from a version with v prefix", installed: "v1.5.0", target: "2.0.0"},
	}
	for _, tt := range validTests {
		t.Run("upgrade "+tt.name+" is valid", func(t *testing.T) {
			err := ValidatePath(tt.installed, tt.target, adrDescriptor)

			assert.NoError(t, err)
		})
	}

	invalidDescriptorTests := []struct {
		name     string
		upgrades []doguv3.Upgrade
	}{
		{name: "unparsable from range", upgrades: []doguv3.Upgrade{{From: ">=a.b", To: "2.0.0"}}},
		{name: "unparsable to version", upgrades: []doguv3.Upgrade{{From: ">=1.0.0 <2.0.0", To: "two"}}},
		{
			name: "overlapping from ranges",
			upgrades: []doguv3.Upgrade{
				{From: ">=1.0.0 <=1.7.0", To: "1.8.0"},
				{From: ">=1.5.0 <2.0.0", To: "2.0.0"},
			},
		},
	}
	for _, tt := range invalidDescriptorTests {
		t.Run("descriptor with "+tt.name+" is invalid", func(t *testing.T) {
			err := ValidatePath("1.5.0", "2.0.0", tt.upgrades)

			assert.ErrorIs(t, err, ErrInvalidDescriptor)
		})
	}
}
