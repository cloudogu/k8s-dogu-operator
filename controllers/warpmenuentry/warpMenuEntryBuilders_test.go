package warpmenuentry

import (
	"strings"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultDoguName           = "test_dogu_name"
	defaultDoguDescriptorName = "testing/" + defaultDoguName
	displayName               = "test_dogu_de"
	warpMenuDisabled          = false
	path                      = "/test_dogu_name"
	doguCategory              = "test_dogu_category"
)

func TestBuildWarpMenuEntry(t *testing.T) {

	t.Run("Should build warp menu entry successfully", func(t *testing.T) {

		doguResource := getDogu(defaultDoguName)
		doguDescriptor := getDoguDescriptor(defaultDoguDescriptorName, displayName, doguCategory, warpTag)
		result := buildWarpMenuEntry(doguDescriptor, doguResource)
		expectedResult := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		assert.Equal(t, expectedResult, result)

	})

	t.Run("Should build warp menu entry successfully", func(t *testing.T) {

		doguResource := getDogu(defaultDoguName)
		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, warpTag)
		result := buildWarpMenuEntry(doguDescriptor, doguResource)
		expectedResult := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		assert.Equal(t, expectedResult, result)

	})
}

func TestDoguShouldBeInWarpMenu(t *testing.T) {
	t.Run("Should return true if warp menu is present in the tags", func(t *testing.T) {

		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, warpTag)
		result := doguShouldBeInWarpMenu(doguDescriptor)
		assert.True(t, result)
	})

	t.Run("Should return false if warp menu is not present in the tags", func(t *testing.T) {

		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, "not"+warpTag)
		result := doguShouldBeInWarpMenu(doguDescriptor)
		assert.False(t, result)
	})
}

func getDoguDescriptor(name, doguDisplayName, category string, tags string) *cesappcore.Dogu {
	return &cesappcore.Dogu{
		Name:        name,
		DisplayName: doguDisplayName,
		Category:    category,
		Tags:        strings.Split(tags, ","),
	}
}

func getDogu(doguName string) *doguv2.Dogu {
	doguResource := &doguv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name: doguName,
		},
	}
	return doguResource
}

func getExpectedWarpMenuEntry(default_dogu_name string, doguResource *doguv2.Dogu, display_name string, warp_menu_disabled bool, path string, dogu_category string) *warpMenuEntryV1.WarpMenuEntry {
	expectedResult := &warpMenuEntryV1.WarpMenuEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name: default_dogu_name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind(kindDogu)),
			},
		},
		Spec: warpMenuEntryV1.WarpMenuEntrySpec{
			DisplayName: warpMenuEntryV1.DisplayName{
				EN: display_name,
				DE: display_name,
			},
			Disabled: warp_menu_disabled,
			Path:     path,
			Category: dogu_category,
		},
	}
	return expectedResult
}
