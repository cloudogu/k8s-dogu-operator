package security

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

func TestValidator_ValidateSecurity(t *testing.T) {
	tests := []struct {
		name           string
		doguDescriptor *core.Dogu
		doguResource   *k8sv2.Dogu
		wantErr        bool
		errMsg         string
	}{
		{
			name: "should succeed",
			doguDescriptor: &core.Dogu{Security: core.Security{Capabilities: core.Capabilities{
				Add:  core.AllCapabilities,
				Drop: core.AllCapabilities,
			}}},
			doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Security: k8sv2.Security{Capabilities: k8sv2.Capabilities{
				Add:  core.AllCapabilities,
				Drop: core.AllCapabilities,
			}}}},
			wantErr: false,
		},
		{
			name: "should fail for dogu descriptor",
			doguDescriptor: &core.Dogu{Security: core.Security{Capabilities: core.Capabilities{
				Add:  []core.Capability{"err"},
				Drop: []core.Capability{"err"},
			}}},
			doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Security: k8sv2.Security{Capabilities: k8sv2.Capabilities{
				Add:  core.AllCapabilities,
				Drop: core.AllCapabilities,
			}}}},
			wantErr: true,
			errMsg:  "invalid security field in dogu descriptor: dogu descriptor : contains at least one invalid security field: err is not a valid capability to be added\nerr is not a valid capability to be dropped",
		},
		{
			name: "should fail for dogu resource",
			doguDescriptor: &core.Dogu{Security: core.Security{Capabilities: core.Capabilities{
				Add:  core.AllCapabilities,
				Drop: core.AllCapabilities,
			}}},
			doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Security: k8sv2.Security{Capabilities: k8sv2.Capabilities{
				Add:  []core.Capability{"err"},
				Drop: []core.Capability{"err"},
			}}}},
			wantErr: true,
			errMsg:  "invalid security field in dogu resource: dogu resource : contains at least one invalid security field: err is not a valid capability to be added\nerr is not a valid capability to be dropped",
		},
		{
			name: "should fail for both",
			doguDescriptor: &core.Dogu{Security: core.Security{Capabilities: core.Capabilities{
				Add:  []core.Capability{"err"},
				Drop: []core.Capability{"err"},
			}}},
			doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Security: k8sv2.Security{Capabilities: k8sv2.Capabilities{
				Add:  []core.Capability{"err"},
				Drop: []core.Capability{"err"},
			}}}},
			wantErr: true,
			errMsg:  "invalid security field in dogu descriptor: dogu descriptor : contains at least one invalid security field: err is not a valid capability to be added\nerr is not a valid capability to be dropped\ninvalid security field in dogu resource: dogu resource : contains at least one invalid security field: err is not a valid capability to be added\nerr is not a valid capability to be dropped",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &validator{}
			err := v.ValidateSecurity(tt.doguDescriptor, tt.doguResource)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			}
		})
	}
}
