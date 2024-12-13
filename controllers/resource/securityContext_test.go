package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestSecurityContextGenerator_Generate(t *testing.T) {
	trueValue := true
	falseValue := false
	profileValue := "myProfile"
	fsGroupValue := int64(10000)
	fsGroupChangePolicyValue := corev1.FSGroupChangeOnRootMismatch
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
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
				AllowPrivilegeEscalation: &falseValue,
			},
		},
		{
			name: "override in dogu resource",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						RunAsNonRoot:           &trueValue,
						ReadOnlyRootFileSystem: &trueValue,
						SELinuxOptions: &v2.SELinuxOptions{
							User:  "myUser",
							Role:  "myRole",
							Type:  "myType",
							Level: "myLevel",
						},
						SeccompProfile: &v2.SeccompProfile{
							Type:             v2.SeccompProfileTypeLocalhost,
							LocalhostProfile: &profileValue,
						},
						AppArmorProfile: &v2.AppArmorProfile{
							Type:             v2.AppArmorProfileTypeLocalhost,
							LocalhostProfile: &profileValue,
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &trueValue,
				SELinuxOptions: &corev1.SELinuxOptions{
					User:  "myUser",
					Role:  "myRole",
					Type:  "myType",
					Level: "myLevel",
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &profileValue,
				},
				AppArmorProfile: &corev1.AppArmorProfile{
					Type:             corev1.AppArmorProfileTypeLocalhost,
					LocalhostProfile: &profileValue,
				},
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &trueValue,
				ReadOnlyRootFilesystem:   &trueValue,
				SELinuxOptions: &corev1.SELinuxOptions{
					User:  "myUser",
					Role:  "myRole",
					Type:  "myType",
					Level: "myLevel",
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &profileValue,
				},
				AppArmorProfile: &corev1.AppArmorProfile{
					Type:             corev1.AppArmorProfileTypeLocalhost,
					LocalhostProfile: &profileValue,
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
							Drop: []v2.Capability{"CHOWN"},
							Add:  []v2.Capability{"AUDIT_READ"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"AUDIT_READ", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
			},
		},
		{
			name: "add 1 new and 1 existing capability",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Add: []v2.Capability{"AUDIT_READ", "DAC_OVERRIDE"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"AUDIT_READ", "CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
			},
		},
		{
			name: "drop all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Drop: []v2.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
			},
		},
		{
			name: "add all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Add: []v2.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"ALL"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
			},
		},
		{
			name: "drop all and add all capabilities",
			dogu: &core.Dogu{},
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Security: v2.Security{
						Capabilities: v2.Capabilities{
							Drop: []v2.Capability{"ALL"},
							Add:  []v2.Capability{"ALL"},
						},
					},
				},
			},
			want1: &corev1.PodSecurityContext{
				RunAsNonRoot: &falseValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"ALL"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
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
				RunAsNonRoot:        &falseValue,
				FSGroup:             &fsGroupValue,
				FSGroupChangePolicy: &fsGroupChangePolicyValue,
			},
			want2: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
					Add:  []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "NET_BIND_SERVICE", "SETGID", "SETPCAP", "SETUID"},
				},
				Privileged:               &falseValue,
				AllowPrivilegeEscalation: &falseValue,
				RunAsNonRoot:             &falseValue,
				ReadOnlyRootFilesystem:   &falseValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SecurityContextGenerator{}
			got1, got2 := s.Generate(tt.dogu, tt.doguResource)
			assert.Equalf(t, tt.want1, got1, "Generate(%v, %v)", tt.dogu, tt.doguResource)
			assert.Equalf(t, tt.want2, got2, "Generate(%v, %v)", tt.dogu, tt.doguResource)
		})
	}
}
