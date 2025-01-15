package resource

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

func TestSecurityContextGenerator_Generate(t *testing.T) {
	tests := []struct {
		name         string
		dogu         *core.Dogu
		doguResource *v2.Dogu
		want1        *corev1.PodSecurityContext
		want2        *corev1.SecurityContext
	}{
		{
			name:         "use dogu defaults",
			dogu:         &core.Dogu{},
			doguResource: &v2.Dogu{},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
			},
		},
		{
			name: "override in dogu resource",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						RunAsNonRoot:           ptr.To(true),
						ReadOnlyRootFileSystem: ptr.To(true),
						SELinuxOptions: &v2.SELinuxOptions{
							User:  "myUser",
							Role:  "myRole",
							Type:  "myType",
							Level: "myLevel",
						},
						SeccompProfile: &v2.SeccompProfile{
							Type:             v2.SeccompProfileTypeLocalhost,
							LocalhostProfile: ptr.To("myProfile"),
						},
						AppArmorProfile: &v2.AppArmorProfile{
							Type:             v2.AppArmorProfileTypeLocalhost,
							LocalhostProfile: ptr.To("myProfile"),
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				SELinuxOptions: &corev1.SELinuxOptions{
					User:  "myUser",
					Role:  "myRole",
					Type:  "myType",
					Level: "myLevel",
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: ptr.To("myProfile"),
				},
				AppArmorProfile: &corev1.AppArmorProfile{
					Type:             corev1.AppArmorProfileTypeLocalhost,
					LocalhostProfile: ptr.To("myProfile"),
				},
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				SELinuxOptions: &corev1.SELinuxOptions{
					User:  "myUser",
					Role:  "myRole",
					Type:  "myType",
					Level: "myLevel",
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: ptr.To("myProfile"),
				},
				AppArmorProfile: &corev1.AppArmorProfile{
					Type:             corev1.AppArmorProfileTypeLocalhost,
					LocalhostProfile: ptr.To("myProfile"),
				},
			},
		},
		{
			name: "drop 1 add 1 capability",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Drop: []core.Capability{"CHOWN"},
							Add:  []core.Capability{"AUDIT_READ"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"AUDIT_READ", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "add 1 new and 1 existing capability",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Add: []core.Capability{"AUDIT_READ", "DAC_OVERRIDE"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"AUDIT_READ", "CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "drop all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Drop: []core.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add:  make([]corev1.Capability, 0),
					Drop: []corev1.Capability{"ALL"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "add all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Add: []core.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add: []corev1.Capability{
						core.AuditControl, core.AuditRead, core.AuditWrite, core.BlockSuspend, core.Bpf, core.CheckpointRestore, core.Chown,
						core.DacOverride, core.Fowner, core.Fsetid, core.IpcLock, core.IpcOwner, core.Kill, core.Lease, core.LinuxImmutable, core.MacAdmin,
						core.MacOverride, core.Mknod, core.NetAdmin, core.NetBindService, core.NetBroadcast, core.NetRaw, core.Perfmon, core.Setfcap,
						core.Setgid, core.Setpcap, core.Setuid, core.Syslog, core.SysAdmin, core.SysBoot, core.SysChroot, core.SysModule, core.SysNice, core.SysPAcct,
						core.SysPTrace, core.SysResource, core.SysTime, core.SysTtyCONFIG, core.WakeAlarm,
					},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "drop all and add all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Drop: []core.Capability{"ALL"},
							Add:  []core.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add: []corev1.Capability{
						core.AuditControl, core.AuditRead, core.AuditWrite, core.BlockSuspend, core.Bpf, core.CheckpointRestore, core.Chown,
						core.DacOverride, core.Fowner, core.Fsetid, core.IpcLock, core.IpcOwner, core.Kill, core.Lease, core.LinuxImmutable, core.MacAdmin,
						core.MacOverride, core.Mknod, core.NetAdmin, core.NetBindService, core.NetBroadcast, core.NetRaw, core.Perfmon, core.Setfcap,
						core.Setgid, core.Setpcap, core.Setuid, core.Syslog, core.SysAdmin, core.SysBoot, core.SysChroot, core.SysModule, core.SysNice, core.SysPAcct,
						core.SysPTrace, core.SysResource, core.SysTime, core.SysTtyCONFIG, core.WakeAlarm,
					},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "dogu with volume",
			dogu: &core.Dogu{
				Volumes: []core.Volume{
					{
						Name:  "myVolume",
						Path:  "/data",
						Owner: "10000",
						Group: "10000",
					},
				},
			},
			doguResource: &v2.Dogu{},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot:        ptr.To(false),
				FSGroup:             ptr.To(int64(10000)),
				FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
		{
			name: "dogu with invalid volume",
			dogu: &core.Dogu{
				Volumes: []core.Volume{
					{
						Name:  "myVolume",
						Path:  "/data",
						Owner: "root",
						Group: "root",
					},
				},
			},
			doguResource: &v2.Dogu{},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(false),
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SecurityContextGenerator{}
			got1, got2 := s.Generate(testCtx, tt.dogu, tt.doguResource)
			slices.Sort(got2.Capabilities.Add)
			slices.Sort(got2.Capabilities.Drop)
			assert.Equalf(t, tt.want1, got1, "Generate(%v, %v)", tt.dogu, tt.doguResource)
			assert.Equalf(t, tt.want2, got2, "Generate(%v, %v)", tt.dogu, tt.doguResource)
		})
	}
}
