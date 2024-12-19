package resource

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

func NewSecurityContextGenerator() *SecurityContextGenerator {
	return &SecurityContextGenerator{}
}

type SecurityContextGenerator struct{}

func (s *SecurityContextGenerator) Generate(dogu *core.Dogu, doguResource *v2.Dogu) (*corev1.PodSecurityContext, *corev1.SecurityContext) {
	runAsNonRoot := isRunAsNonRoot(dogu, doguResource)
	seLinuxOptions := seLinuxOptions(doguResource.Spec.Security.SELinuxOptions)
	appArmorProfile := appArmorProfile(doguResource.Spec.Security.AppArmorProfile)
	seccompProfile := seccompProfile(doguResource.Spec.Security.SeccompProfile)

	readOnlyRootFS := isReadOnlyRootFS(dogu, doguResource)
	// We never want those to be true and don't respect the dogu descriptor's privileged flag which is deprecated anyway.
	privileged := false
	allowPrivilegeEscalation := false

	fsGroup, fsGroupChangePolicy := fsGroupAndChangePolicy(dogu)

	return &corev1.PodSecurityContext{
			RunAsNonRoot:        &runAsNonRoot,
			SELinuxOptions:      seLinuxOptions,
			AppArmorProfile:     appArmorProfile,
			SeccompProfile:      seccompProfile,
			FSGroup:             fsGroup,
			FSGroupChangePolicy: fsGroupChangePolicy,
		}, &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
				Add:  effectiveCapabilities(dogu, doguResource),
			},
			ReadOnlyRootFilesystem:   &readOnlyRootFS,
			RunAsNonRoot:             &runAsNonRoot,
			SELinuxOptions:           seLinuxOptions,
			AppArmorProfile:          appArmorProfile,
			SeccompProfile:           seccompProfile,
			Privileged:               &privileged,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		}
}

func fsGroupAndChangePolicy(dogu *core.Dogu) (*int64, *corev1.PodFSGroupChangePolicy) {
	if len(dogu.Volumes) > 0 {
		fsGroupChangePolicy := corev1.FSGroupChangeOnRootMismatch
		group, _ := strconv.Atoi(dogu.Volumes[0].Group)
		gid := int64(group)

		return &gid, &fsGroupChangePolicy
	}

	return nil, nil
}

func seccompProfile(profile *v2.SeccompProfile) *corev1.SeccompProfile {
	if profile == nil {
		return nil
	}

	return &corev1.SeccompProfile{
		Type:             corev1.SeccompProfileType(profile.Type),
		LocalhostProfile: profile.LocalhostProfile,
	}
}

func appArmorProfile(profile *v2.AppArmorProfile) *corev1.AppArmorProfile {
	if profile == nil {
		return nil
	}

	return &corev1.AppArmorProfile{
		Type:             corev1.AppArmorProfileType(profile.Type),
		LocalhostProfile: profile.LocalhostProfile,
	}
}

func seLinuxOptions(options *v2.SELinuxOptions) *corev1.SELinuxOptions {
	if options == nil {
		return nil
	}

	return &corev1.SELinuxOptions{
		User:  options.User,
		Role:  options.Role,
		Type:  options.Type,
		Level: options.Level,
	}
}

func isRunAsNonRoot(dogu *core.Dogu, resource *v2.Dogu) bool {
	if resource.Spec.Security.RunAsNonRoot != nil {
		return *resource.Spec.Security.RunAsNonRoot
	}

	return dogu.Security.RunAsNonRoot
}

func isReadOnlyRootFS(dogu *core.Dogu, resource *v2.Dogu) bool {
	if resource.Spec.Security.ReadOnlyRootFileSystem != nil {
		return *resource.Spec.Security.ReadOnlyRootFileSystem
	}

	return dogu.Security.ReadOnlyRootFileSystem
}

func effectiveCapabilities(dogu *core.Dogu, doguResource *v2.Dogu) []corev1.Capability {
	doguDescriptorCapabilities := dogu.EffectiveCapabilities()
	effectiveCapabilities := make(map[corev1.Capability]struct{}, len(doguDescriptorCapabilities))
	for _, capability := range doguDescriptorCapabilities {
		effectiveCapabilities[corev1.Capability(capability)] = struct{}{}
	}

	for _, dropCap := range doguResource.Spec.Security.Capabilities.Drop {
		if dropCap == "ALL" {
			effectiveCapabilities = make(map[corev1.Capability]struct{})
			break
		}
		delete(effectiveCapabilities, corev1.Capability(dropCap))
	}

	for _, addCap := range doguResource.Spec.Security.Capabilities.Add {
		if addCap == "ALL" {
			return []corev1.Capability{"ALL"}
		}
		effectiveCapabilities[corev1.Capability(addCap)] = struct{}{}
	}

	capabilitiesSlice := slices.Collect(maps.Keys(effectiveCapabilities))
	slices.SortFunc(capabilitiesSlice, func(a, b corev1.Capability) int {
		return strings.Compare(string(a), string(b))
	})
	return capabilitiesSlice
}
