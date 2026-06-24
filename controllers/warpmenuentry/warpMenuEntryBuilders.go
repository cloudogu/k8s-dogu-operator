package warpmenuentry

import (
	"strings"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kindDogu = "Dogu"
	warpTag  = "warp"
)

func buildWarpMenuEntry(doguDescriptor *cesappcore.Dogu, doguResource *doguv2.Dogu) *warpMenuEntryV1.WarpMenuEntry {

	warpMenuEntry := &warpMenuEntryV1.WarpMenuEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name: doguResource.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind(kindDogu)),
			},
		},
		Spec: warpMenuEntryV1.WarpMenuEntrySpec{
			DisplayName: warpMenuEntryV1.DisplayName{
				EN: doguDescriptor.DisplayName,
				DE: doguDescriptor.DisplayName,
			},
			Disabled: false,
			Path:     createPath(doguDescriptor.Name),
			Category: doguDescriptor.Category,
		},
	}
	return warpMenuEntry
}

func createPath(name string) string {
	// remove namespace
	parts := strings.Split(name, "/")
	return "/" + parts[len(parts)-1]
}

func doguShouldBeInWarpMenu(doguDescriptor *cesappcore.Dogu) bool {
	return containsString(doguDescriptor.Tags, warpTag)
}

// ContainsString returns true if the slice contains the item
func containsString(slice []string, item string) bool {
	for _, sliceItem := range slice {
		if sliceItem == item {
			return true
		}
	}
	return false
}
