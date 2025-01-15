package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var testCtx = context.TODO()

func Test_evaluateRequiredOperation(t *testing.T) {
	t.Run("installed should return upgrade", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv2.DoguSpec{Name: "official/ledogu", Version: "9000.0.0-1"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			recorder: recorder,
			fetcher:  localDoguFetcher,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Upgrade}, operations)
	})
	t.Run("installed should return no operations for any other changes on a pre-existing dogu resource", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv2.DoguSpec{Name: "official/ledogu", Version: "42.0.0-1"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			recorder: recorder,
			fetcher:  localDoguFetcher,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Empty(t, operations)
	})
	t.Run("installed should fail because of version parsing errors", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ledogu"},
			Spec:       k8sv2.DoguSpec{Name: "official/ledogu", Version: "lol.I.don't.care-äöüß"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)
		recorder.On("Eventf", testDoguCr, v1.EventTypeWarning, operatorEventReason, mock.Anything, mock.Anything)
		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			recorder: recorder,
			fetcher:  localDoguFetcher,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse major version")
		assert.Empty(t, operations)
	})

	t.Run("deletiontimestamp should return delete", func(t *testing.T) {
		// given
		now := metav1.NewTime(time.Now())
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ledogu",
				DeletionTimestamp: &now,
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		sut := &doguReconciler{}

		// when
		operations, err := sut.evaluateRequiredOperations(nil, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Delete}, operations)
		testDoguCr.DeletionTimestamp = nil
	})

	t.Run("installing should return wait", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{Name: "official/ledogu", Version: "42.0.0-1"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalling,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Wait}, operations)
	})

	t.Run("upgrading should return wait", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{Name: "official/ledogu", Version: "42.0.0-1"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusUpgrading,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Wait}, operations)
	})

	t.Run("installed with changed ingress annotation should return IngressAnnotationChange", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:                         "official/ledogu",
				Version:                      "42.0.0-1",
				AdditionalIngressAnnotations: map[string]string{"annotation1": "value1"},
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{ChangeAdditionalIngressAnnotations}, operations)
	})

	t.Run("installed with changed security should return ChangeSecurityContext", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:    "official/ledogu",
				Version: "42.0.0-1",
				Security: k8sv2.Security{
					RunAsNonRoot: ptr.To(true),
				},
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{ChangeSecurityContext}, operations)
	})

	t.Run("check for ingress annotations should fail", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:    "official/ledogu",
				Version: "42.0.0-1",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{
			Name:        "ledogu",
			Annotations: map[string]string{annotation.AdditionalIngressAnnotationsAnnotation: "{{\"invalid json"},
		}}
		fakeClient := fake.NewClientBuilder().WithObjects(doguService).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(nil, testDoguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get additional ingress annotations from service of dogu [ledogu]")
		assert.Nil(t, operations)
	})

	t.Run("check for security should fail", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:    "official/ledogu",
				Version: "42.0.0-1",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalled,
			},
		}

		recorder := newMockEventRecorder(t)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		fakeClient := fake.NewClientBuilder().WithObjects(doguService).Build()

		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(nil, assert.AnError)

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if security context is changed")
		assert.Nil(t, operations)
	})

	t.Run("installing with changed ingress annotation should return Wait and IngressAnnotationChange", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:                         "official/ledogu",
				Version:                      "42.0.0-1",
				AdditionalIngressAnnotations: map[string]string{"annotation1": "value1"},
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalling,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Wait, ChangeAdditionalIngressAnnotations}, operations)
	})

	t.Run("installing with changed security should return Wait and ChangeSecurityContext", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:    "official/ledogu",
				Version: "42.0.0-1",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusInstalling,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Wait, ChangeSecurityContext}, operations)
	})

	t.Run("pvc resizing with changed ingress annotation should return PVCResize, IngressAnnotationChange", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:                         "official/ledogu",
				Version:                      "42.0.0-1",
				AdditionalIngressAnnotations: map[string]string{"annotation1": "value1"},
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusPVCResizing,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{ExpandVolume, ChangeAdditionalIngressAnnotations}, operations)
	})

	t.Run("pvc resizing with changed security should return PVCResize, ChangeSecurityContext", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{
				Name:    "official/ledogu",
				Version: "42.0.0-1",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusPVCResizing,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{ExpandVolume, ChangeSecurityContext}, operations)
	})

	t.Run("deleting should return no operations", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusDeleting,
			},
		}

		sut := &doguReconciler{}

		// when
		operations, err := sut.evaluateRequiredOperations(nil, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Empty(t, operations)
	})

	t.Run("not installed should return install", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusNotInstalled,
			},
		}

		sut := &doguReconciler{}

		// when
		operations, err := sut.evaluateRequiredOperations(nil, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{Install}, operations)
	})

	t.Run("pvc resizing should return expand volume", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Spec: k8sv2.DoguSpec{Name: "official/ledogu", Version: "42.0.0-1"},
			Status: k8sv2.DoguStatus{
				Status: k8sv2.DoguStatusPVCResizing,
			},
		}

		recorder := newMockEventRecorder(t)

		localDogu := &core.Dogu{Name: "official/ledogu", Version: "42.0.0-1"}
		localDoguFetcher := newMockLocalDoguFetcher(t)
		localDoguFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(localDogu, nil)

		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}}
		doguDeployment := newDoguDeploymentWithSecurity(
			&v1.PodSecurityContext{
				SELinuxOptions:  nil,
				RunAsNonRoot:    ptr.To(false),
				SeccompProfile:  nil,
				AppArmorProfile: nil,
			},
			&v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []v1.Capability{core.All},
				},
				Privileged:               ptr.To(false),
				SELinuxOptions:           nil,
				RunAsNonRoot:             ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(false),
				AllowPrivilegeEscalation: ptr.To(false),
				SeccompProfile:           nil,
				AppArmorProfile:          nil,
			},
		)
		fakeClient := fake.NewClientBuilder().WithObjects(doguService, doguDeployment).Build()

		sut := &doguReconciler{
			client:   fakeClient,
			fetcher:  localDoguFetcher,
			recorder: recorder,
		}

		// when
		operations, err := sut.evaluateRequiredOperations(testCtx, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, []operation{ExpandVolume}, operations)
	})

	t.Run("default should return no operations", func(t *testing.T) {
		// given
		testDoguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledogu",
			},
			Status: k8sv2.DoguStatus{
				Status: "youaresomethingelse",
			},
		}

		sut := &doguReconciler{}

		// when
		operations, err := sut.evaluateRequiredOperations(nil, testDoguCr)

		// then
		require.NoError(t, err)
		assert.Empty(t, operations)
	})
}

func Test_doguResourceChangeDebugPredicate_Update(t *testing.T) {
	oldDoguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Generation: 123456789},
		Spec:       k8sv2.DoguSpec{Name: "ns/dogu", Version: "1.2.3-4"}}
	newDoguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Generation: 987654321},
		Spec:       k8sv2.DoguSpec{Name: "ns/dogu", Version: "1.2.3-5"}}

	t.Run("should should return false for dogu installation", func(t *testing.T) {
		recorder := newMockEventRecorder(t)
		recorder.On("Event", newDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: nil,
			ObjectNew: newDoguResource,
		})

		// then
		require.False(t, actual)
	})
	t.Run("should should return false for dogu deletion", func(t *testing.T) {
		recorder := newMockEventRecorder(t)
		recorder.On("Event", oldDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: nil,
		})

		// then
		require.False(t, actual)
	})
	t.Run("should should return true for dogu upgrade", func(t *testing.T) {
		recorder := newMockEventRecorder(t)
		recorder.On("Event", newDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: newDoguResource,
		})

		// then
		require.True(t, actual)
	})
	t.Run("should should return false for no dogu change", func(t *testing.T) {
		recorder := newMockEventRecorder(t)
		recorder.On("Event", oldDoguResource, "Normal", "Debug", mock.Anything)
		sut := doguResourceChangeDebugPredicate{recorder: recorder}

		// when
		actual := sut.Update(event.UpdateEvent{
			ObjectOld: oldDoguResource,
			ObjectNew: oldDoguResource,
		})

		// then
		require.False(t, actual)
	})
}

func Test_buildResourceDiff(t *testing.T) {
	oldDoguResource := &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Name: "ns/dogu", Version: "1.2.3-4"}}
	newDoguResource := &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Name: "ns/dogu", Version: "1.2.3-5"}}

	type args struct {
		objOld client.Object
		objNew client.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "create-diff",
			args: args{objOld: nil, objNew: newDoguResource},
			want: "  any(\n+ \t&v2.Dogu{Spec: v2.DoguSpec{Name: \"ns/dogu\", Version: \"1.2.3-5\"}},\n  )\n",
		},
		{
			name: "upgrade-diff",
			args: args{objOld: oldDoguResource, objNew: newDoguResource},
			want: "  &v2.Dogu{\n  \tTypeMeta:   {},\n  \tObjectMeta: {},\n  \tSpec: v2.DoguSpec{\n  \t\tName:      \"ns/dogu\",\n- \t\tVersion:   \"1.2.3-4\",\n+ \t\tVersion:   \"1.2.3-5\",\n  \t\tResources: {},\n  \t\tSecurity:  {},\n  \t\t... // 4 identical fields\n  \t},\n  \tStatus: {},\n  }\n",
		},
		{
			name: "delete-diff",
			args: args{objOld: oldDoguResource, objNew: nil},
			want: "  any(\n- \t&v2.Dogu{Spec: v2.DoguSpec{Name: \"ns/dogu\", Version: \"1.2.3-4\"}},\n  )\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := buildResourceDiff(tt.args.objOld, tt.args.objNew)
			assert.Equalf(t,
				tt.want,
				result,
				"buildResourceDiff(%v, %v)", tt.args.objOld, tt.args.objNew)
		})
	}
}

func Test_finishOperation(t *testing.T) {
	result, err := finishOperation()

	assert.Empty(t, result)
	assert.Nil(t, err)
}

func Test_requeueOrFinishOperation(t *testing.T) {
	input := ctrl.Result{
		Requeue: true,
	}

	result, err := requeueOrFinishOperation(input)

	assert.Equal(t, input, result)
	assert.Nil(t, err)
}

func Test_requeueWithError(t *testing.T) {
	result, err := requeueWithError(assert.AnError)

	assert.Empty(t, result)
	assert.Same(t, assert.AnError, err)
}

func Test_operation_toString(t *testing.T) {
	assert.Equal(t, operation("Install"), Install)
	assert.Equal(t, operation("Upgrade"), Upgrade)
	assert.Equal(t, operation("Delete"), Delete)
}

func Test_doguReconciler_checkForVolumeExpansion(t *testing.T) {
	t.Run("should return false and nil if no pvc is found", func(t *testing.T) {
		// given
		sut := &doguReconciler{client: fake.NewClientBuilder().Build()}
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return false and nil if pvc is found but dogu has no dataVolumeSize property", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta()}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return error on invalid volume size", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "wrong"}}}
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta()}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse resource volume size")
		assert.False(t, expand)
	})

	t.Run("should return true if volume size is higher than actual", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("1Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.VolumeResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.True(t, expand)
	})

	t.Run("should return false if size is equal", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("2Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.VolumeResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.False(t, expand)
	})

	t.Run("should return error if size is smaller than actual", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2Gi"}}}
		resources := make(map[v1.ResourceName]resource.Quantity)
		resources[v1.ResourceStorage] = resource.MustParse("3Gi")
		pvc := &v1.PersistentVolumeClaim{ObjectMeta: *doguCr.GetObjectMeta(),
			Spec: v1.PersistentVolumeClaimSpec{Resources: v1.VolumeResourceRequirements{Requests: resources}}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(pvc).Build()}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid dogu state for dogu [test] as requested volume size is "+
			"[2Gi] while existing volume is [3Gi], shrinking of volumes is not allowed")
		assert.False(t, expand)
	})

	t.Run("error on pvc found", func(t *testing.T) {
		// given
		sut := &doguReconciler{client: fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()}
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}

		// when
		expand, err := sut.checkForVolumeExpansion(testCtx, doguCr)

		// then
		require.Error(t, err)
		assert.False(t, expand)
	})
}

func Test_doguReconciler_checkForAdditionalIngressAnnotations(t *testing.T) {
	t.Run("should return true if annotations are not euqal", func(t *testing.T) {
		// given
		doguIngressAnnotation := map[string]string{"test": "test"}
		serviceIngressAnnotation := map[string]string{"sdf": "sdfsdf"}
		marshalServiceAnnotations, err := json.Marshal(serviceIngressAnnotation)
		require.NoError(t, err)
		annotationsService := map[string]string{
			"k8s-dogu-operator.cloudogu.com/additional-ingress-annotations": string(marshalServiceAnnotations),
		}
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec: k8sv2.DoguSpec{AdditionalIngressAnnotations: doguIngressAnnotation}}
		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test", Annotations: annotationsService}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(doguService).Build()}

		// when
		result, err := sut.checkForAdditionalIngressAnnotations(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("should return error if service annotations are not map[string]string", func(t *testing.T) {
		// given
		serviceIngressAnnotation := map[string]bool{"sdf": true}
		marshalServiceAnnotations, err := json.Marshal(serviceIngressAnnotation)
		require.NoError(t, err)
		annotationsService := map[string]string{
			"k8s-dogu-operator.cloudogu.com/additional-ingress-annotations": string(marshalServiceAnnotations),
		}
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test", Annotations: annotationsService}}
		sut := &doguReconciler{client: fake.NewClientBuilder().WithObjects(doguService).Build()}

		// when
		_, err = sut.checkForAdditionalIngressAnnotations(testCtx, doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get additional ingress annotations from service of dogu [test]")
	})

	t.Run("should return error if no service is found", func(t *testing.T) {
		// given
		doguCr := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
		sut := &doguReconciler{client: fake.NewClientBuilder().Build()}

		// when
		_, err := sut.checkForAdditionalIngressAnnotations(testCtx, doguCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service of dogu [test]")
	})
}

func Test_doguReconciler_validateSpecName(t *testing.T) {
	tests := []struct {
		name         string
		recorderFunc func(t *testing.T) record.EventRecorder
		doguResource *k8sv2.Dogu
		wantSuccess  bool
	}{
		{
			name: "should fail validation",
			recorderFunc: func(t *testing.T) record.EventRecorder {
				recorder := newMockEventRecorder(t)
				recorder.EXPECT().Eventf(mock.Anything, "Warning", "FailedNameValidation", "Dogu resource does not follow naming rules: The dogu's simple name '%s' must be the same as the resource name '%s'.", "invalid-example", "example")
				return recorder
			},
			doguResource: &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "example"}, Spec: k8sv2.DoguSpec{Name: "testing/invalid-example"}},
			wantSuccess:  false,
		},
		{
			name:         "should succeed validation",
			recorderFunc: func(t *testing.T) record.EventRecorder { return newMockEventRecorder(t) },
			doguResource: &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "example"}, Spec: k8sv2.DoguSpec{Name: "testing/example"}},
			wantSuccess:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &doguReconciler{recorder: tt.recorderFunc(t)}
			assert.Equal(t, tt.wantSuccess, r.validateName(tt.doguResource))
		})
	}
}

func Test_doguReconciler_executeRequiredOperation(t *testing.T) {
	t.Run("should finish if no operation required", func(t *testing.T) {
		// given
		sut := &doguReconciler{}
		var requiredOperations []operation
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusInstalled}}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should finish if only wait is required", func(t *testing.T) {
		// given
		sut := &doguReconciler{}
		requiredOperations := []operation{Wait}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusInstalling}}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should requeue if wait and another operation are required", func(t *testing.T) {
		// given
		sut := &doguReconciler{}
		requiredOperations := []operation{Wait, ExpandVolume}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusInstalling}}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, actual)
	})
	t.Run("should install", func(t *testing.T) {
		// given
		requiredOperations := []operation{Install}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Install(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "Installation", "%s successful.", "Installation").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to install dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should requeue on install error", func(t *testing.T) {
		// given
		requiredOperations := []operation{Install}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Install(testCtx, doguResource).Return(assert.AnError)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Warning", "ErrInstallation", "%s failed. Reason: %s.", "Installation", assert.AnError.Error()).Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to install dogu ldap", doguResource, assert.AnError, mock.Anything).Return(ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, actual)
	})
	t.Run("should fail to handle install error", func(t *testing.T) {
		// given
		requiredOperations := []operation{Install}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Install(testCtx, doguResource).Return(assert.AnError)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Warning", "ErrInstallation", "%s failed. Reason: %s.", "Installation", assert.AnError.Error()).Return()
		mockRecorder.EXPECT().Eventf(doguResource, "Warning", "ErrRequeue", "Failed to requeue the %s.", "installation").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to install dogu ldap", doguResource, assert.AnError, mock.Anything).Return(ctrl.Result{}, assert.AnError)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should install", func(t *testing.T) {
		// given
		requiredOperations := []operation{Install}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Install(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "Installation", "%s successful.", "Installation").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to install dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should upgrade", func(t *testing.T) {
		// given
		requiredOperations := []operation{Upgrade}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Upgrade(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "Upgrading", "%s successful.", "Upgrade").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to upgrade dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should delete", func(t *testing.T) {
		// given
		requiredOperations := []operation{Delete}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().Delete(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "Deinstallation", "%s successful.", "Deinstallation").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to delete dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should expand volume", func(t *testing.T) {
		// given
		requiredOperations := []operation{ExpandVolume}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().SetDoguDataVolumeSize(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "VolumeExpansion", "%s successful.", "VolumeExpansion").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to expand volume dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should change additional ingress annotations", func(t *testing.T) {
		// given
		requiredOperations := []operation{ChangeAdditionalIngressAnnotations}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().SetDoguAdditionalIngressAnnotations(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "AdditionalIngressAnnotationsChange", "%s successful.", "AdditionalIngressAnnotationsChange").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to change additional ingress annotations dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should finish for other operations", func(t *testing.T) {
		// given
		requiredOperations := []operation{operation("some_operation")}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		sut := &doguReconciler{}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should requeue on multiple operations", func(t *testing.T) {
		// given
		requiredOperations := []operation{ExpandVolume, AdditionalIngressAnnotationsChangeEventReason}
		doguResource := &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "ecosystem",
		}, Status: k8sv2.DoguStatus{Status: k8sv2.DoguStatusNotInstalled}}

		mockDoguManager := NewMockCombinedDoguManager(t)
		mockDoguManager.EXPECT().SetDoguDataVolumeSize(testCtx, doguResource).Return(nil)
		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Eventf(doguResource, "Normal", "VolumeExpansion", "%s successful.", "VolumeExpansion").Return()
		mockRequeueHandler := newMockRequeueHandler(t)
		mockRequeueHandler.EXPECT().Handle(testCtx, "failed to expand volume dogu ldap", doguResource, nil, mock.Anything).Return(ctrl.Result{}, nil)
		sut := &doguReconciler{
			doguManager:        mockDoguManager,
			recorder:           mockRecorder,
			doguRequeueHandler: mockRequeueHandler,
		}

		// when
		actual, err := sut.executeRequiredOperation(testCtx, requiredOperations, doguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true}, actual)
	})
}

func Test_doguReconciler_validateVolumeSize(t *testing.T) {
	type args struct {
		doguResource *k8sv2.Dogu
	}
	tests := []struct {
		name         string
		args         args
		recorderFunc func(t *testing.T) record.EventRecorder
		wantSuccess  bool
	}{
		{
			name:         "success with Binary-SI",
			args:         args{doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2Gi"}}}},
			recorderFunc: func(t *testing.T) record.EventRecorder { return newMockEventRecorder(t) },
			wantSuccess:  true,
		},
		{
			name: "should fail on invalid size",
			args: args{doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2invalidGi"}}}},
			recorderFunc: func(t *testing.T) record.EventRecorder {
				recorder := newMockEventRecorder(t)
				recorder.EXPECT().Eventf(mock.Anything, "Warning", "FailedVolumeSizeParsingValidation", "Dogu resource volume size parsing error: %s", "2invalidGi")
				return recorder
			},
			wantSuccess: false,
		},
		{
			name: "should fail on non Binary-SI",
			args: args{doguResource: &k8sv2.Dogu{Spec: k8sv2.DoguSpec{Resources: k8sv2.DoguResources{DataVolumeSize: "2G"}}}},
			recorderFunc: func(t *testing.T) record.EventRecorder {
				recorder := newMockEventRecorder(t)
				recorder.EXPECT().Eventf(mock.Anything, "Warning", "FailedVolumeSizeSIValidation", "Dogu resource volume size format is not Binary-SI (\"Mi\" or \"Gi\"): %s", resource.MustParse("2G"))
				return recorder
			},
			wantSuccess: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &doguReconciler{
				recorder: tt.recorderFunc(t),
			}
			assert.Equalf(t, tt.wantSuccess, r.validateVolumeSize(tt.args.doguResource), "validateVolumeSize(%v)", tt.args.doguResource)
		})
	}
}

func Test_doguReconciler_checkSecurityContextChanged(t *testing.T) {
	tests := []struct {
		name         string
		deployment   *appsv1.Deployment
		doguResource *k8sv2.Dogu
		fetcherFn    func(t *testing.T) localDoguFetcher
		want         bool
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "failed to get dogu descriptor",
			doguResource: &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}},
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(nil, assert.AnError)
				return fetcher
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) && assert.ErrorContains(t, err, "failed to get dogu descriptor \"ledogu\"", i)
			},
		},
		{
			name:         "failed to get dogu deployment",
			doguResource: &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}},
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to get deployment of dogu \"ledogu\"", i)
			},
		},
		{
			name: "pod security context changed fs group",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					FSGroup:             ptr.To(int64(55)),
					FSGroupChangePolicy: ptr.To(v1.FSGroupChangeOnRootMismatch),
					SELinuxOptions:      &v1.SELinuxOptions{},
					RunAsNonRoot:        ptr.To(true),
					SeccompProfile:      &v1.SeccompProfile{},
					AppArmorProfile:     &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{
					Volumes: []core.Volume{{Group: "10001"}},
				}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "pod security context changed fs group change policy",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					FSGroup:             ptr.To(int64(10001)),
					FSGroupChangePolicy: ptr.To(v1.FSGroupChangeAlways),
					SELinuxOptions:      &v1.SELinuxOptions{},
					RunAsNonRoot:        ptr.To(true),
					SeccompProfile:      &v1.SeccompProfile{},
					AppArmorProfile:     &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{
					Volumes: []core.Volume{{Group: "10001"}},
				}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "pod security context changed Run as Non Root",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(false),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "pod security context changed SeccompProfile",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions: &v1.SELinuxOptions{},
					RunAsNonRoot:   ptr.To(true),
					SeccompProfile: &v1.SeccompProfile{
						Type:             v1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("myProfile"),
					},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "pod security context changed AppArmorProfile",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions: &v1.SELinuxOptions{},
					RunAsNonRoot:   ptr.To(true),
					SeccompProfile: &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{
						Type:             v1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("myProfile"),
					},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "pod security context changed SELinuxOptions",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions: &v1.SELinuxOptions{
						User:  "user",
						Role:  "role",
						Type:  "type",
						Level: "level",
					},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed Run as Non Root",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed ReadOnlyRootFilesystem",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed SeccompProfile",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile: &v1.SeccompProfile{
						Type:             v1.SeccompProfileTypeLocalhost,
						LocalhostProfile: ptr.To("myProfile"),
					},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed AppArmorProfile",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{
						Type:             v1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To("myProfile"),
					},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed SELinuxOptions",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged: ptr.To(false),
					SELinuxOptions: &v1.SELinuxOptions{
						User:  "user",
						Role:  "role",
						Type:  "type",
						Level: "level",
					},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "container security context changed Capabilities",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add:  []core.Capability{core.All},
					Drop: []core.Capability{core.Chown, core.DacOverride},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "security context not changed",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  &v1.SELinuxOptions{},
					RunAsNonRoot:    ptr.To(true),
					SeccompProfile:  &v1.SeccompProfile{},
					AppArmorProfile: &v1.AppArmorProfile{},
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           &v1.SELinuxOptions{},
					RunAsNonRoot:             ptr.To(true),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &v1.SeccompProfile{},
					AppArmorProfile:          &v1.AppArmorProfile{},
				},
			),
			doguResource: newDoguResourceWithSecurity(
				true,
				false,
				&k8sv2.SeccompProfile{},
				&k8sv2.AppArmorProfile{},
				&k8sv2.SELinuxOptions{},
				k8sv2.Capabilities{
					Add: []core.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
						core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
					Drop: []core.Capability{core.All},
				},
			),
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    false,
			wantErr: assert.NoError,
		},
		{
			name: "security context not changed with descriptor defaults",
			deployment: newDoguDeploymentWithSecurity(
				&v1.PodSecurityContext{
					SELinuxOptions:  nil,
					RunAsNonRoot:    ptr.To(false),
					SeccompProfile:  nil,
					AppArmorProfile: nil,
				},
				&v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{core.Chown, core.DacOverride, core.Fowner, core.Fsetid,
							core.Kill, core.NetBindService, core.Setgid, core.Setpcap, core.Setuid},
						Drop: []v1.Capability{core.All},
					},
					Privileged:               ptr.To(false),
					SELinuxOptions:           nil,
					RunAsNonRoot:             ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(false),
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           nil,
					AppArmorProfile:          nil,
				},
			),
			doguResource: &k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "ledogu"}},
			fetcherFn: func(t *testing.T) localDoguFetcher {
				fetcher := newMockLocalDoguFetcher(t)
				fetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ledogu")).Return(&core.Dogu{}, nil)
				return fetcher
			},
			want:    false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fakeClient client.Client
			if tt.deployment != nil {
				fakeClient = fake.NewClientBuilder().WithObjects(tt.deployment).Build()
			} else {
				fakeClient = fake.NewClientBuilder().Build()
			}
			r := &doguReconciler{
				client:  fakeClient,
				fetcher: tt.fetcherFn(t),
			}
			got, err := r.checkSecurityContextChanged(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("checkSecurityContextChanged(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "checkSecurityContextChanged(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func newDoguResourceWithSecurity(runAsNonRoot bool, readOnlyRootFileSystem bool, seccompProfile *k8sv2.SeccompProfile, appArmorProfile *k8sv2.AppArmorProfile, seLinuxOptions *k8sv2.SELinuxOptions, capabilities k8sv2.Capabilities) *k8sv2.Dogu {
	return &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ledogu",
		},
		Spec: k8sv2.DoguSpec{
			Security: k8sv2.Security{
				RunAsNonRoot:           ptr.To(runAsNonRoot),
				ReadOnlyRootFileSystem: ptr.To(readOnlyRootFileSystem),
				SeccompProfile:         seccompProfile,
				AppArmorProfile:        appArmorProfile,
				SELinuxOptions:         seLinuxOptions,
				Capabilities:           capabilities,
			},
		},
	}
}

func newDoguDeploymentWithSecurity(podSecurityContext *v1.PodSecurityContext, containerSecurityContext *v1.SecurityContext) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ledogu",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					SecurityContext: podSecurityContext,
					Containers:      []v1.Container{{SecurityContext: containerSecurityContext}},
				},
			},
		},
	}
}
