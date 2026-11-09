package controllers

import (
	"context"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func volumeLabelTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newVolumeLabelFakeClient(funcs interceptor.Funcs, objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(volumeLabelTestScheme()).
		WithObjects(objects...).
		WithInterceptorFuncs(funcs).
		Build()
}

func volumeLabelTestStatefulSet(name string, labels map[string]string, templates ...string) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{Name: name, Namespace: testNamespace, Labels: labels}
	for _, template := range templates {
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{Name: template})
	}
	return sts
}

// withClaimMounts makes the StatefulSet's pod template mount the given claims.
func withClaimMounts(statefulSet *appsv1.StatefulSet, claimNames ...string) *appsv1.StatefulSet {
	statefulSet.Spec.Template = volumeLabelTestPodTemplate(claimNames...)
	return statefulSet
}

func volumeLabelTestPodTemplate(claimNames ...string) corev1.PodTemplateSpec {
	template := corev1.PodTemplateSpec{}
	for _, claimName := range claimNames {
		template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
			Name:                  claimName,
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		})
	}
	return template
}

func volumeLabelTestDeployment(name string, labels map[string]string, claimNames ...string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{Name: name, Namespace: testNamespace, Labels: labels}
	deployment.Spec.Template = volumeLabelTestPodTemplate(claimNames...)
	return deployment
}

func volumeLabelTestDaemonSet(name string, labels map[string]string, claimNames ...string) *appsv1.DaemonSet {
	daemonSet := &appsv1.DaemonSet{Name: name, Namespace: testNamespace, Labels: labels}
	daemonSet.Spec.Template = volumeLabelTestPodTemplate(claimNames...)
	return daemonSet
}

func persistedVolumeLabels(t *testing.T, c client.Client, name string) map[string]string {
	t.Helper()
	persisted := &corev1.PersistentVolumeClaim{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, persisted))
	return persisted.Labels
}

func TestNewDoguVolumeLabelReconciler(t *testing.T) {
	t.Run("should register the controller with the manager if dogu v3 is enabled", func(t *testing.T) {
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(volumeLabelTestScheme())
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(nil)
		managerMock.EXPECT().GetCache().Return(nil)
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		sut, err := NewDoguVolumeLabelReconciler(nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should not register the controller with the manager if dogu v3 is disabled", func(t *testing.T) {
		// the manager mock fails the test on any call
		sut, err := NewDoguVolumeLabelReconciler(nil, newMockCtrlManager(t), &opConfig.OperatorConfig{DoguV3Enabled: false})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should return error if the controller cannot be registered", func(t *testing.T) {
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(volumeLabelTestScheme())
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(assert.AnError)
		managerMock.EXPECT().GetCache().Return(nil).Maybe()
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		_, err := NewDoguVolumeLabelReconciler(nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to setup dogu-v3-volume-labels controller")
	})
}

func TestDoguVolumeLabelReconciler_Reconcile(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "ldap"}
	casLabels := map[string]string{v3beta1.DoguLabelName: "cas"}
	ldapVolumeLabels := map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap"}

	type reconcileCase struct {
		name      string
		workloads []client.Object
		volume    *corev1.PersistentVolumeClaim
		// nil: no write, labels unchanged
		want map[string]string
	}
	tests := []reconcileCase{
		{
			name:      "should add both dogu labels to a claim template volume",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace},
			want:      ldapVolumeLabels,
		},
		{
			name:      "should add both dogu labels to a volume mounted by a deployment",
			workloads: []client.Object{volumeLabelTestDeployment("ldap", doguLabels, "existing")},
			volume:    &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace},
			want:      ldapVolumeLabels,
		},
		{
			name:      "should add both dogu labels to a volume mounted by a daemonset",
			workloads: []client.Object{volumeLabelTestDaemonSet("ldap", doguLabels, "existing")},
			volume:    &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace},
			want:      ldapVolumeLabels,
		},
		{
			name:      "should add both dogu labels to a volume mounted by a statefulset pod template",
			workloads: []client.Object{withClaimMounts(volumeLabelTestStatefulSet("ldap", doguLabels), "existing")},
			volume:    &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace},
			want:      ldapVolumeLabels,
		},
		{
			name: "should label a volume mounted and generated from a claim template by the same dogu",
			workloads: []client.Object{
				volumeLabelTestStatefulSet("ldap", doguLabels, "data"),
				volumeLabelTestDeployment("ldap-backup", doguLabels, "data-ldap-0"),
			},
			volume: &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace},
			want:   ldapVolumeLabels,
		},
		{
			name:      "should overwrite a wrong dogu label value and keep other labels",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace, Labels: map[string]string{v3beta1.DoguLabelName: "cas", "app": "ces"}},
			want:      map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap", "app": "ces"},
		},
		{
			name:      "should label a volume carrying the backup scope",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace, Labels: map[string]string{"k8s.cloudogu.com/backup-scope": "misc"}},
			want:      map[string]string{v3beta1.DoguLabelName: "ldap", doguv2.DoguLabelName: "ldap", "k8s.cloudogu.com/backup-scope": "misc"},
		},
		{
			name: "should ignore a matching statefulset without dogu label",
			// the volume name matches the claim templates of both StatefulSets
			workloads: []client.Object{
				volumeLabelTestStatefulSet("b-c", map[string]string{"app": "ces"}, "a"),
				volumeLabelTestStatefulSet("c", doguLabels, "a-b"),
			},
			volume: &corev1.PersistentVolumeClaim{Name: "a-b-c-0", Namespace: testNamespace},
			// does not include `app: ces`-label
			want: ldapVolumeLabels,
		},
		{
			name: "should leave a volume unchanged that matches claim templates of two dogus",
			// the volume name matches the claim templates of both StatefulSets
			workloads: []client.Object{
				volumeLabelTestStatefulSet("b-c", doguLabels, "a"),
				volumeLabelTestStatefulSet("c", casLabels, "a-b"),
			},
			volume: &corev1.PersistentVolumeClaim{Name: "a-b-c-0", Namespace: testNamespace},
			want:   nil,
		},
		{
			name: "should leave a volume unchanged that is mounted by workloads of two dogus",
			workloads: []client.Object{
				volumeLabelTestDeployment("ldap", doguLabels, "shared"),
				volumeLabelTestDaemonSet("cas", casLabels, "shared"),
			},
			volume: &corev1.PersistentVolumeClaim{Name: "shared", Namespace: testNamespace},
			want:   nil,
		},
		{
			name:      "should leave a volume unchanged that is mounted by a deployment without dogu label",
			workloads: []client.Object{volumeLabelTestDeployment("ldap", map[string]string{"app": "ces"}, "existing")},
			volume:    &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace},
			want:      nil,
		},
		{
			name:      "should leave a volume unchanged that is being deleted",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume: &corev1.PersistentVolumeClaim{
				Name:              "data-ldap-0",
				Namespace:         testNamespace,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{"kubernetes.io/pvc-protection"},
			},
			want: nil,
		},
		{
			name:      "should not write a volume whose dogu labels are already correct",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace, Labels: ldapVolumeLabels},
			want:      nil,
		},
		{
			name:      "should leave a volume unchanged that matches no claim template",
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: "foreign", Namespace: testNamespace, Labels: map[string]string{"app": "foreign"}},
			want:      nil,
		},
	}
	for _, name := range []string{"data-ldap-01", "data-ldap-x", "data-ldap-", "other-ldap-0", "data-cas-0", "data-ldap-0-1"} {
		tests = append(tests, reconcileCase{
			name:      "should not match volume " + name,
			workloads: []client.Object{volumeLabelTestStatefulSet("ldap", doguLabels, "data")},
			volume:    &corev1.PersistentVolumeClaim{Name: name, Namespace: testNamespace},
			want:      nil,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcs := interceptor.Funcs{}
			want := tt.want
			if want == nil {
				// expect no patch
				funcs = interceptor.Funcs{
					Patch: func(ctx context.Context, _ client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						t.Errorf("unexpected patch of %q", obj.GetName())
						return nil
					},
				}
				want = tt.volume.Labels
			}
			c := newVolumeLabelFakeClient(funcs, append(tt.workloads, tt.volume)...)
			sut := newDoguVolumeLabelReconciler(c)

			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tt.volume)})

			require.NoError(t, err)
			assert.Equal(t, reconcile.Result{}, result)
			assert.Equal(t, want, persistedVolumeLabels(t, c, tt.volume.Name))
		})
	}

	t.Run("should do nothing if the volume does not exist", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{}, volumeLabelTestStatefulSet("ldap", doguLabels, "data"))
		sut := newDoguVolumeLabelReconciler(c)

		// Request with non-existent volume-name
		result, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("should return error if the volume cannot be fetched", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			Get: func(ctx context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			},
		})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get volume")
	})

	t.Run("should return error if the statefulsets cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, _ client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			},
		}, &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list dogu statefulsets")
	})

	t.Run("should return error if the deployments cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, wc client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*appsv1.DeploymentList); ok {
					return assert.AnError
				}
				return wc.List(ctx, list, opts...)
			},
		}, &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "existing"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list dogu deployments")
	})

	t.Run("should return error if the daemonsets cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, wc client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*appsv1.DaemonSetList); ok {
					return assert.AnError
				}
				return wc.List(ctx, list, opts...)
			},
		}, &corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "existing"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list dogu daemonsets")
	})

	t.Run("should return error if the volume cannot be patched", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			Patch: func(ctx context.Context, _ client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				return assert.AnError
			},
		}, volumeLabelTestStatefulSet("ldap", doguLabels, "data"), &corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace})
		sut := newDoguVolumeLabelReconciler(c)

		_, err := sut.Reconcile(context.Background(), reconcile.Request{Namespace: testNamespace, Name: "data-ldap-0"})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to patch dogu labels")
	})
}

func TestNewDoguVolumeRequestMapper(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "ldap"}
	deploymentWithEmptyDir := volumeLabelTestDeployment("ldap", doguLabels, "other")
	deploymentWithEmptyDir.Spec.Template.Spec.Volumes = append(deploymentWithEmptyDir.Spec.Template.Spec.Volumes,
		// Regression case: volume name differs from its claim name, so mapping the volume name instead of the claim would fail
		corev1.Volume{Name: "data", PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "existing"}},
		corev1.Volume{Name: "tmp", EmptyDir: &corev1.EmptyDirVolumeSource{}},
	)

	tests := []struct {
		name     string
		workload client.Object
		volumes  []client.Object
		want     []string
	}{
		{
			name:     "should map a statefulset to the existing volumes of its claim templates",
			workload: volumeLabelTestStatefulSet("ldap", doguLabels, "data", "logs"),
			volumes: []client.Object{
				&corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace},
				&corev1.PersistentVolumeClaim{Name: "data-ldap-1", Namespace: testNamespace},
				&corev1.PersistentVolumeClaim{Name: "logs-ldap-0", Namespace: testNamespace},
				&corev1.PersistentVolumeClaim{Name: "data-cas-0", Namespace: testNamespace},
				&corev1.PersistentVolumeClaim{Name: "data-ldap-01", Namespace: testNamespace},
			},
			want: []string{"data-ldap-0", "data-ldap-1", "logs-ldap-0"},
		},
		{
			name:     "should map a statefulset to its mounted volumes and the existing volumes of its claim templates",
			workload: withClaimMounts(volumeLabelTestStatefulSet("ldap", doguLabels, "data"), "existing"),
			volumes:  []client.Object{&corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace}},
			want:     []string{"existing", "data-ldap-0"},
		},
		{
			// mounted volumes are mapped without a lookup, so the volume need not exist yet
			name:     "should map a deployment to its mounted volumes and skip non-PVCs",
			workload: deploymentWithEmptyDir,
			want:     []string{"existing", "other"},
		},
		{
			name:     "should map a daemonset to its mounted volumes",
			workload: volumeLabelTestDaemonSet("ldap", doguLabels, "existing"),
			want:     []string{"existing"},
		},
		{
			name:     "should map a deployment without volumes to nothing",
			workload: volumeLabelTestDeployment("ldap", doguLabels),
			volumes:  []client.Object{&corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace}},
			want:     nil,
		},
		{
			name:     "should map a daemonset without volumes to nothing",
			workload: volumeLabelTestDaemonSet("ldap", doguLabels),
			volumes:  []client.Object{&corev1.PersistentVolumeClaim{Name: "existing", Namespace: testNamespace}},
			want:     nil,
		},
		{
			name:     "should map a statefulset without claim templates to nothing",
			workload: volumeLabelTestStatefulSet("ldap", doguLabels),
			volumes:  []client.Object{&corev1.PersistentVolumeClaim{Name: "data-ldap-0", Namespace: testNamespace}},
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newVolumeLabelFakeClient(interceptor.Funcs{}, tt.volumes...)

			requests := newDoguVolumeRequestMapper(c)(context.Background(), tt.workload)

			var want []reconcile.Request
			for _, name := range tt.want {
				want = append(want, reconcile.Request{Namespace: testNamespace, Name: name})
			}
			assert.ElementsMatch(t, want, requests)
		})
	}

	t.Run("should map to nothing if the volumes cannot be listed", func(t *testing.T) {
		c := newVolumeLabelFakeClient(interceptor.Funcs{
			List: func(ctx context.Context, _ client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return assert.AnError
			},
		})

		requests := newDoguVolumeRequestMapper(c)(context.Background(), volumeLabelTestStatefulSet("ldap", doguLabels, "data"))

		assert.Empty(t, requests)
	})
}

func TestDoguWorkloadChangedPredicate(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "ldap"}
	statefulSet := func(generation int64, labels map[string]string, readyReplicas int32) *appsv1.StatefulSet {
		sts := volumeLabelTestStatefulSet("ldap", labels, "data")
		sts.Generation = generation
		sts.Status.ReadyReplicas = readyReplicas
		return sts
	}

	updates := []struct {
		name   string
		old    *appsv1.StatefulSet
		new    *appsv1.StatefulSet
		passes bool
	}{
		{name: "should pass a spec change", old: statefulSet(1, doguLabels, 1), new: statefulSet(2, doguLabels, 1), passes: true},
		{name: "should pass a label change", old: statefulSet(1, doguLabels, 1), new: statefulSet(1, map[string]string{v3beta1.DoguLabelName: "cas"}, 1), passes: true},
		{name: "should drop a status-only change", old: statefulSet(1, doguLabels, 0), new: statefulSet(1, doguLabels, 1), passes: false},
		{name: "should drop a spec change of a workload without dogu label", old: statefulSet(1, nil, 1), new: statefulSet(2, nil, 1), passes: false},
	}
	for _, tt := range updates {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.passes, doguWorkloadChangedPredicate().Update(event.UpdateEvent{ObjectOld: tt.old, ObjectNew: tt.new}))
		})
	}

	t.Run("should pass create and delete of a dogu workload", func(t *testing.T) {
		sut := doguWorkloadChangedPredicate()

		assert.True(t, sut.Create(event.CreateEvent{Object: statefulSet(1, doguLabels, 0)}))
		assert.True(t, sut.Delete(event.DeleteEvent{Object: statefulSet(1, doguLabels, 0)}))
	})
}
