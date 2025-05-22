package resource

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

func NewSecurityContextGenerator() *SecurityContextGenerator {
	return &SecurityContextGenerator{}
}

// SecurityContextGenerator provides functionality to create security contexts for dogus.
type SecurityContextGenerator struct{}

// Generate creates security contexts for the pod and containers of a dogu.
func (s *SecurityContextGenerator) Generate(ctx context.Context, dogu *core.Dogu, doguResource *v2.Dogu) (*corev1.PodSecurityContext, *corev1.SecurityContext) {
	runAsNonRoot := isRunAsNonRoot(dogu, doguResource)
	seLinuxOptions := seLinuxOptions(doguResource.Spec.Security.SELinuxOptions)
	appArmorProfile := appArmorProfile(doguResource.Spec.Security.AppArmorProfile)
	seccompProfile := seccompProfile(doguResource.Spec.Security.SeccompProfile)

	readOnlyRootFS := isReadOnlyRootFS(dogu, doguResource)
	// We never want those to be true and don't respect the dogu descriptor's privileged flag which is deprecated anyway.
	privileged := false

	fsGroup, fsGroupChangePolicy := fsGroupAndChangePolicy(ctx, dogu)

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
			ReadOnlyRootFilesystem: &readOnlyRootFS,
			RunAsNonRoot:           &runAsNonRoot,
			SELinuxOptions:         seLinuxOptions,
			AppArmorProfile:        appArmorProfile,
			SeccompProfile:         seccompProfile,
			Privileged:             &privileged,
		}
}

func fsGroupAndChangePolicy(ctx context.Context, dogu *core.Dogu) (*int64, *corev1.PodFSGroupChangePolicy) {
	if len(dogu.Volumes) > 0 {
		rawGroup := dogu.Volumes[0].Group
		group, err := strconv.Atoi(rawGroup)
		if err != nil {
			// this only happens if the dogu descriptor is invalid; not much we can do here
			// maybe consider using int64 instead of string for the group in the dogu-descriptor?
			log.FromContext(ctx).Error(err, fmt.Sprintf("dogu-descriptor %q: failed to convert group %q in volume to int", dogu.Name, rawGroup))
			return nil, nil
		}

		return ptr.To(int64(group)), ptr.To(corev1.FSGroupChangeOnRootMismatch)
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
	return ptr.Deref(resource.Spec.Security.RunAsNonRoot, dogu.Security.RunAsNonRoot)
}

func isReadOnlyRootFS(dogu *core.Dogu, resource *v2.Dogu) bool {
	return ptr.Deref(resource.Spec.Security.ReadOnlyRootFileSystem, dogu.Security.ReadOnlyRootFileSystem)
}

func effectiveCapabilities(dogu *core.Dogu, doguResource *v2.Dogu) []corev1.Capability {
	effectiveCapabilities := core.CalcEffectiveCapabilities(
		dogu.EffectiveCapabilities(),
		doguResource.Spec.Security.Capabilities.Drop,
		doguResource.Spec.Security.Capabilities.Add,
	)

	effectiveK8sCapabilities := make([]corev1.Capability, len(effectiveCapabilities))
	for i, capability := range effectiveCapabilities {
		effectiveK8sCapabilities[i] = corev1.Capability(capability)
	}
	return effectiveK8sCapabilities
}
