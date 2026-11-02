package migration

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3beta1ac "github.com/cloudogu/k8s-dogu-lib/v3/applyconfiguration/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestMigrateDoguPVCStepWithSharedClient(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: "ecosystem"}}
	dogu.Spec.Name = "nexus"
	labels := map[string]string{"dogu.name": "nexus"}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: dogu.Namespace, Labels: labels},
		Spec:       appsv1.DeploymentSpec{Replicas: new(int32(1))},
		Status:     appsv1.DeploymentStatus{Replicas: 1},
	}
	otherDeployment := deployment.DeepCopy()
	otherDeployment.Namespace = "other"
	oldPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "old-claim", Namespace: dogu.Namespace, Labels: labels},
		Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "existing-volume"},
	}
	otherPVC := oldPVC.DeepCopy()
	otherPVC.Namespace = "other"
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "existing-volume"},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			ClaimRef:                      &corev1.ObjectReference{Name: oldPVC.Name, Namespace: dogu.Namespace},
			StorageClassName:              "existing-class",
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity:                      corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
			VolumeMode:                    new(corev1.PersistentVolumeFilesystem),
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dogu, &corev1.PersistentVolumeClaim{}).
		WithObjects(dogu, deployment, otherDeployment, oldPVC, otherPVC, pv).Build()
	step := NewMigrateDoguPVCStep(k8sClient, record.NewFakeRecorder(10))

	// Scale down only the deployment in the dogu namespace.
	result := step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.Equal(t, requeueTime, result.RequeueAfter)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment))
	assert.Zero(t, *deployment.Spec.Replicas)
	deployment.Status.Replicas = 0
	require.NoError(t, k8sClient.Status().Update(ctx, deployment))

	// Persist the PV name from the PVC before changing the volume or deleting the claim.
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.Equal(t, requeueTime, result.RequeueAfter)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(dogu), dogu))
	assert.Equal(t, pv.Name, dogu.Annotations[migrationSourcePVAnnotation])
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(oldPVC), oldPVC))
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(pv), pv))
	assert.Equal(t, corev1.PersistentVolumeReclaimDelete, pv.Spec.PersistentVolumeReclaimPolicy)

	// Retain the PV before deleting the old PVC by its metadata name.
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(pv), pv))
	assert.Equal(t, corev1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.True(t, apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(oldPVC), oldPVC)))
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(otherPVC), otherPVC))
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(otherDeployment), otherDeployment))
	assert.EqualValues(t, 1, *otherDeployment.Spec.Replicas)

	// Simulate a restart: recover the source PV solely from the persisted annotation.
	dogu = &v3beta1.Dogu{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Name: "nexus", Namespace: "ecosystem"}, dogu))
	recorder := record.NewFakeRecorder(10)
	step = NewMigrateDoguPVCStep(k8sClient, recorder)

	// Reserve the PV before creating the migration claim.
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(pv), pv))
	require.NotNil(t, pv.Spec.ClaimRef)
	assert.Equal(t, "nexus-migration", pv.Spec.ClaimRef.Name)
	assert.Equal(t, dogu.Namespace, pv.Spec.ClaimRef.Namespace)
	assert.Empty(t, pv.Spec.ClaimRef.UID)
	assert.True(t, apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKey{Name: "nexus-migration", Namespace: dogu.Namespace}, &corev1.PersistentVolumeClaim{})))
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.False(t, result.Continue)
	assert.Equal(t, requeueTime, result.RequeueAfter)
	migrationPVC := &corev1.PersistentVolumeClaim{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: "nexus-migration"}, migrationPVC))
	assert.Equal(t, pv.Name, migrationPVC.Spec.VolumeName)
	require.NotNil(t, migrationPVC.Spec.StorageClassName)
	assert.Equal(t, pv.Spec.StorageClassName, *migrationPVC.Spec.StorageClassName)
	assert.Equal(t, pv.Spec.AccessModes, migrationPVC.Spec.AccessModes)
	assert.Equal(t, pv.Spec.Capacity, migrationPVC.Spec.Resources.Requests)
	assert.Equal(t, pv.Spec.VolumeMode, migrationPVC.Spec.VolumeMode)
	assert.Nil(t, meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionReady))
	// Creating the migration claim is the only point that emits the event.
	require.Len(t, recorder.Events, 1)
	assert.Equal(t, "Normal MigrationPending Prepared the dogu pvc for the v3 migration", <-recorder.Events)

	// Waiting for the binding must not emit the event again.
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.False(t, result.Continue)
	assert.Equal(t, requeueTime, result.RequeueAfter)
	assert.Empty(t, recorder.Events)

	// The API assigns the UID; the volume controller confirms the binding asynchronously.
	migrationPVC.UID = types.UID("migration-pvc-uid")
	require.NoError(t, k8sClient.Update(ctx, migrationPVC))
	migrationPVC.Status.Phase = corev1.ClaimBound
	require.NoError(t, k8sClient.Status().Update(ctx, migrationPVC))
	pv.Spec.ClaimRef.UID = migrationPVC.UID
	require.NoError(t, k8sClient.Update(ctx, pv))
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.True(t, result.Continue)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(pv), pv))
	require.NotNil(t, pv.Spec.ClaimRef)
	assert.Equal(t, migrationPVC.Name, pv.Spec.ClaimRef.Name)
	assert.Equal(t, migrationPVC.UID, pv.Spec.ClaimRef.UID)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(dogu), dogu))
	condition := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionReady)
	assert.Nil(t, condition, "completed binding must skip status updates")
	assert.Empty(t, recorder.Events)

	// Repeating the completed step preserves the binding.
	result = step.Run(ctx, dogu)
	require.NoError(t, result.Err)
	assert.True(t, result.Continue)
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(pv), pv))
	assert.Equal(t, migrationPVC.UID, pv.Spec.ClaimRef.UID)
}

func TestMigrateDoguPVCStepSourcePVDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name       string
		claimCount int
		volumeName string
		applyErr   bool
		errMessage string
	}{
		{name: "missing source PVC", errMessage: "found 0 source PVCs"},
		{name: "ambiguous source PVCs", claimCount: 2, volumeName: "source-pv", errMessage: "found 2 source PVCs"},
		{name: "unbound source PVC", claimCount: 1},
		{name: "annotation apply fails", claimCount: 1, volumeName: "source-pv", applyErr: true, errMessage: "could not persist migration source PV"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, appsv1.AddToScheme(scheme))
			require.NoError(t, v3beta1.AddToScheme(scheme))
			dogu := &v3beta1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: "ecosystem", Annotations: map[string]string{"existing": "preserved"}}}
			dogu.Spec.Name = "nexus"
			labels := map[string]string{"dogu.name": "nexus"}
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: dogu.Namespace, Labels: labels},
				Spec:       appsv1.DeploymentSpec{Replicas: new(int32(0))},
			}
			objects := []client.Object{dogu, deployment}
			for i := 0; i < tt.claimCount; i++ {
				objects = append(objects, &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: string(rune('a' + i)), Namespace: dogu.Namespace, Labels: labels},
					Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: tt.volumeName},
				})
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).
				WithInterceptorFuncs(interceptor.Funcs{
					Apply: func(ctx context.Context, c client.WithWatch, configuration runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
						if tt.applyErr {
							return errors.New("apply rejected")
						}
						return c.Apply(ctx, configuration, opts...)
					},
				}).Build()
			step := NewMigrateDoguPVCStep(k8sClient, record.NewFakeRecorder(10))
			result := step.Run(t.Context(), dogu)
			assert.False(t, result.Continue)
			assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
			if tt.errMessage != "" {
				require.ErrorContains(t, result.Err, tt.errMessage)
			} else {
				require.NoError(t, result.Err)
				assert.Equal(t, requeueTime, result.RequeueAfter)
			}
			assert.Empty(t, dogu.Annotations[migrationSourcePVAnnotation])
			storedDogu := &v3beta1.Dogu{}
			require.NoError(t, k8sClient.Get(t.Context(), client.ObjectKeyFromObject(dogu), storedDogu))
			assert.Equal(t, map[string]string{"existing": "preserved"}, storedDogu.Annotations)
			claims := &corev1.PersistentVolumeClaimList{}
			require.NoError(t, k8sClient.List(t.Context(), claims, client.InNamespace(dogu.Namespace)))
			assert.Len(t, claims.Items, tt.claimCount)
		})
	}
}

// The fixture starts after the volume controller has completed the migration
// binding. Tests move it back to the phase they exercise.
type pvcMigrationFixture struct {
	dogu       *v3beta1.Dogu
	deployment *appsv1.Deployment
	pv         *corev1.PersistentVolume
	claim      *corev1.PersistentVolumeClaim
	client     client.WithWatch
	recorder   *record.FakeRecorder
	step       *MigrateDoguPVCStep
}

func newPVCMigrationFixture(t *testing.T) *pvcMigrationFixture {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	f := &pvcMigrationFixture{
		dogu: &v3beta1.Dogu{ObjectMeta: metav1.ObjectMeta{
			Name: "nexus", Namespace: "ecosystem", Generation: 3,
			Annotations: map[string]string{migrationSourcePVAnnotation: "source-pv", "existing": "preserved"},
		}},
		deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: "ecosystem", Labels: map[string]string{"dogu.name": "nexus"}},
			Spec:       appsv1.DeploymentSpec{Replicas: new(int32(0))},
		},
		pv: &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "source-pv"},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				ClaimRef:                      &corev1.ObjectReference{APIVersion: "v1", Kind: "PersistentVolumeClaim", Name: "nexus-migration", Namespace: "ecosystem", UID: "migration-uid"},
				AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Capacity:                      corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
			},
		},
		claim: &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "nexus-migration", Namespace: "ecosystem", UID: "migration-uid"},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "source-pv"},
			Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		},
		recorder: record.NewFakeRecorder(10),
	}
	f.dogu.Spec.Name = "nexus"
	f.client = fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(f.dogu, f.claim).
		WithObjects(f.dogu, f.deployment, f.pv, f.claim).Build()
	for _, obj := range []client.Object{f.dogu, f.deployment, f.pv, f.claim} {
		require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(obj), obj))
	}
	f.step = NewMigrateDoguPVCStep(f.client, f.recorder)
	return f
}

func TestMigrateDoguPVCStepCheckMigrationStatus(t *testing.T) {
	for _, tt := range []struct {
		name     string
		prepare  func(*pvcMigrationFixture)
		complete bool
	}{
		{name: "confirmed binding", complete: true},
		{name: "missing annotations", prepare: func(f *pvcMigrationFixture) { f.dogu.Annotations = nil }},
		{name: "missing source annotation", prepare: func(f *pvcMigrationFixture) { delete(f.dogu.Annotations, migrationSourcePVAnnotation) }},
		{name: "missing migration PVC", prepare: func(f *pvcMigrationFixture) { f.claim = nil }},
		{name: "missing source PV", prepare: func(f *pvcMigrationFixture) { f.pv = nil }},
		{name: "pending PVC", prepare: func(f *pvcMigrationFixture) { f.claim.Status.Phase = corev1.ClaimPending }},
		{name: "lost PVC", prepare: func(f *pvcMigrationFixture) { f.claim.Status.Phase = corev1.ClaimLost }},
		{name: "deleting PVC", prepare: func(f *pvcMigrationFixture) {
			f.claim.DeletionTimestamp = new(metav1.Now())
			f.claim.Finalizers = []string{"kubernetes.io/pvc-protection"}
		}},
		{name: "different volume", prepare: func(f *pvcMigrationFixture) { f.claim.Spec.VolumeName = "other-pv" }},
		{name: "missing volume", prepare: func(f *pvcMigrationFixture) { f.claim.Spec.VolumeName = "" }},
		{name: "missing PVC UID", prepare: func(f *pvcMigrationFixture) { f.claim.UID = "" }},
		{name: "missing claim reference", prepare: func(f *pvcMigrationFixture) { f.pv.Spec.ClaimRef = nil }},
		{name: "different claim name", prepare: func(f *pvcMigrationFixture) { f.pv.Spec.ClaimRef.Name = "other-claim" }},
		{name: "different namespace", prepare: func(f *pvcMigrationFixture) { f.pv.Spec.ClaimRef.Namespace = "other" }},
		{name: "reservation without UID", prepare: func(f *pvcMigrationFixture) { f.pv.Spec.ClaimRef.UID = "" }},
		{name: "different claim UID", prepare: func(f *pvcMigrationFixture) { f.pv.Spec.ClaimRef.UID = "old-uid" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			if tt.prepare != nil {
				tt.prepare(f)
			}
			builder := fake.NewClientBuilder().WithScheme(f.client.Scheme())
			if f.claim != nil {
				builder.WithObjects(f.claim)
			}
			if f.pv != nil {
				builder.WithObjects(f.pv)
			}
			f.step.client = builder.Build()

			complete, err := f.step.checkMigrationStatus(t.Context(), f.dogu)
			require.NoError(t, err)
			assert.Equal(t, tt.complete, complete)
		})
	}
}

func TestMigrateDoguPVCStepCompletedMigrationIsNoOp(t *testing.T) {
	f := newPVCMigrationFixture(t)
	originalDogu := f.dogu.DeepCopy()
	// Only the two reads needed to confirm the binding are allowed. Any other
	// client operation calls the nil embedded client and fails the test.
	reads := []client.ObjectKey{}
	f.step.client = interceptor.NewClient(nil, interceptor.Funcs{
		Get: func(ctx context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			reads = append(reads, key)
			return f.client.Get(ctx, key, obj, opts...)
		},
	})

	for range 2 {
		result := f.step.Run(t.Context(), f.dogu)
		assert.Equal(t, stepsv3.Continue(), result)
	}
	assert.Equal(t, []client.ObjectKey{
		client.ObjectKeyFromObject(f.claim), client.ObjectKeyFromObject(f.pv),
		client.ObjectKeyFromObject(f.claim), client.ObjectKeyFromObject(f.pv),
	}, reads)
	assert.Equal(t, originalDogu, f.dogu)
	assert.Empty(t, f.recorder.Events)
}

func TestMigrateDoguPVCStepCheckMigrationStatusAPIErrors(t *testing.T) {
	for _, resource := range []client.Object{&corev1.PersistentVolumeClaim{}, &corev1.PersistentVolume{}} {
		t.Run(reflect.TypeOf(resource).Elem().Name(), func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			apiErr := errors.New("API request rejected")
			failed := false
			f.step.client = interceptor.NewClient(nil, interceptor.Funcs{
				Get: func(ctx context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					require.False(t, failed, "no further reads after a failure")
					if reflect.TypeOf(obj) == reflect.TypeOf(resource) {
						failed = true
						return apiErr
					}
					return f.client.Get(ctx, key, obj, opts...)
				},
			})

			complete, err := f.step.checkMigrationStatus(t.Context(), f.dogu)
			assert.False(t, complete)
			require.ErrorIs(t, err, apiErr)
			failed = false
			result := f.step.Run(t.Context(), f.dogu)
			require.True(t, failed)
			require.ErrorIs(t, result.Err, apiErr)
			assert.False(t, result.Continue)
			assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
			assert.Empty(t, f.recorder.Events)
		})
	}
}

func TestMigrateDoguPVCStepBindingConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		phase    corev1.PersistentVolumeClaimPhase
		claimUID types.UID
		refUID   types.UID
		complete bool
	}{
		{name: "pending", phase: corev1.ClaimPending, claimUID: "migration-uid", refUID: "migration-uid"},
		{name: "lost", phase: corev1.ClaimLost, claimUID: "migration-uid", refUID: "migration-uid"},
		{name: "reservation without binding UID", phase: corev1.ClaimBound, claimUID: "migration-uid"},
		{name: "binding to an earlier claim of the same name", phase: corev1.ClaimBound, claimUID: "migration-uid", refUID: "old-uid"},
		{name: "missing claim UID", phase: corev1.ClaimBound},
		{name: "confirmed binding", phase: corev1.ClaimBound, claimUID: "migration-uid", refUID: "migration-uid", complete: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			f.claim.UID = tt.claimUID
			require.NoError(t, f.client.Update(t.Context(), f.claim))
			f.claim.Status.Phase = tt.phase
			require.NoError(t, f.client.Status().Update(t.Context(), f.claim))
			f.pv.Spec.ClaimRef.UID = tt.refUID
			require.NoError(t, f.client.Update(t.Context(), f.pv))

			result := f.step.ensureMigrationPVCBound(t.Context(), f.dogu, f.pv)
			require.NoError(t, result.Err)
			assert.Equal(t, tt.complete, result.Continue)
			if !tt.complete {
				assert.Equal(t, requeueTime, result.RequeueAfter)
				assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
			}
			// The event is only emitted when the claim is created, never while confirming the binding.
			assert.Empty(t, f.recorder.Events)
		})
	}
}

func TestMigrateDoguPVCStepPVReservation(t *testing.T) {
	for _, tt := range []struct {
		name string
		ref  *corev1.ObjectReference
	}{
		{name: "unreserved PV"},
		{name: "old source claim", ref: &corev1.ObjectReference{Name: "old-claim", Namespace: "ecosystem", UID: "old-uid"}},
		{name: "matching claim name in a different namespace", ref: &corev1.ObjectReference{Name: "nexus-migration", Namespace: "other", UID: "old-uid"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			require.NoError(t, f.client.Delete(t.Context(), f.claim))
			f.pv.Spec.ClaimRef = tt.ref
			require.NoError(t, f.client.Update(t.Context(), f.pv))
			updates := 0
			f.step.client = interceptor.NewClient(f.client, interceptor.Funcs{
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if pv, ok := obj.(*corev1.PersistentVolume); ok {
						updates++
						require.NotNil(t, pv.Spec.ClaimRef, "never expose an unreserved PV")
						assert.Equal(t, "nexus-migration", pv.Spec.ClaimRef.Name)
						assert.Equal(t, "ecosystem", pv.Spec.ClaimRef.Namespace)
						assert.Empty(t, pv.Spec.ClaimRef.UID)
					}
					return c.Update(ctx, obj, opts...)
				},
			})
			result := f.step.Run(t.Context(), f.dogu)
			require.NoError(t, result.Err)
			assert.False(t, result.Continue)
			assert.Equal(t, requeueTime, result.RequeueAfter)
			assert.Equal(t, 1, updates)
			require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(f.pv), f.pv))
			require.NotNil(t, f.pv.Spec.ClaimRef)
			assert.Equal(t, "nexus-migration", f.pv.Spec.ClaimRef.Name)
			assert.True(t, apierrors.IsNotFound(f.client.Get(t.Context(), client.ObjectKeyFromObject(f.claim), f.claim)))
			assert.Empty(t, f.recorder.Events)

			// The next reconciliation creates the claim while keeping the reservation.
			result = f.step.Run(t.Context(), f.dogu)
			require.NoError(t, result.Err)
			assert.False(t, result.Continue)
			assert.Equal(t, 1, updates)
			require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(f.claim), f.claim))
			assert.Equal(t, f.pv.Name, f.claim.Spec.VolumeName)
			require.NotNil(t, f.claim.Spec.StorageClassName)
			assert.Empty(t, *f.claim.Spec.StorageClassName, "preserve an explicit empty storage class")
			require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(f.pv), f.pv))
			assert.Equal(t, "nexus-migration", f.pv.Spec.ClaimRef.Name)
			require.Len(t, f.recorder.Events, 1)
			assert.Equal(t, "Normal MigrationPending Prepared the dogu pvc for the v3 migration", <-f.recorder.Events)
		})
	}
}

func TestMigrateDoguPVCStepPersistSourcePVName(t *testing.T) {
	t.Run("reuse saved name without querying deleted source claims", func(t *testing.T) {
		f := newPVCMigrationFixture(t)
		f.step.client = interceptor.NewClient(f.client, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				t.Fatal("persisted source PV must not require another source PVC lookup")
				return nil
			},
		})
		name, result := f.step.persistSourcePVName(t.Context(), f.dogu)
		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
		assert.Equal(t, f.pv.Name, name)
	})
	t.Run("preserve existing annotations when saving source PV", func(t *testing.T) {
		f := newPVCMigrationFixture(t)
		delete(f.dogu.Annotations, migrationSourcePVAnnotation)
		require.NoError(t, f.client.Update(t.Context(), f.dogu))
		source := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: "ecosystem", Labels: map[string]string{"dogu.name": "nexus"}},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: f.pv.Name},
		}
		require.NoError(t, f.client.Create(t.Context(), source))
		originalDogu := f.dogu.DeepCopy()
		_, result := f.step.persistSourcePVName(t.Context(), f.dogu)
		require.NoError(t, result.Err)
		assert.False(t, result.Continue)
		assert.Equal(t, requeueTime, result.RequeueAfter)
		assert.Equal(t, originalDogu, f.dogu, "apply must not replace the local dogu resource")
		require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), f.dogu))
		assert.Equal(t, map[string]string{migrationSourcePVAnnotation: f.pv.Name, "existing": "preserved"}, f.dogu.Annotations)
		assert.Equal(t, originalDogu.Spec, f.dogu.Spec, "applying the annotation must preserve the spec")
	})
}

func TestMigrateDoguPVCStepDeploymentCount(t *testing.T) {
	for _, count := range []int{0, 2} {
		t.Run(string(rune('0'+count))+" deployments", func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			require.NoError(t, f.client.Delete(t.Context(), f.claim))
			if count == 0 {
				require.NoError(t, f.client.Delete(t.Context(), f.deployment))
			} else {
				other := f.deployment.DeepCopy()
				other.Name = "second-deployment"
				other.ResourceVersion = ""
				require.NoError(t, f.client.Create(t.Context(), other))
			}
			result := f.step.Run(t.Context(), f.dogu)
			require.ErrorContains(t, result.Err, "expected exactly one")
			assert.False(t, result.Continue)
			assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
			assert.Empty(t, f.recorder.Events)
		})
	}
}

func TestMigrateDoguPVCStepAPIErrors(t *testing.T) {
	addSourceClaim := func(t *testing.T, f *pvcMigrationFixture) {
		t.Helper()
		require.NoError(t, f.client.Create(t.Context(), &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: "ecosystem", Labels: map[string]string{"dogu.name": "nexus"}},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: f.pv.Name},
		}))
	}
	discoverSource := func(t *testing.T, f *pvcMigrationFixture) {
		t.Helper()
		delete(f.dogu.Annotations, migrationSourcePVAnnotation)
		require.NoError(t, f.client.Update(t.Context(), f.dogu))
		addSourceClaim(t, f)
	}
	for _, tt := range []struct {
		name      string
		operation string
		resource  any
		prepare   func(*testing.T, *pvcMigrationFixture)
		message   string
	}{
		{name: "list deployments", operation: "list", resource: &appsv1.DeploymentList{}, message: "could not list v2 dogu deployment"},
		{name: "scale deployment", operation: "update", resource: &appsv1.Deployment{}, message: "could not scale down v2 deployment", prepare: func(t *testing.T, f *pvcMigrationFixture) {
			f.deployment.Spec.Replicas = new(int32(1))
			require.NoError(t, f.client.Update(t.Context(), f.deployment))
			f.deployment.Status.Replicas = 1
			require.NoError(t, f.client.Status().Update(t.Context(), f.deployment))
		}},
		{name: "discover source claim", operation: "list", resource: &corev1.PersistentVolumeClaimList{}, prepare: discoverSource, message: "could not list source PVCs"},
		{name: "save source annotation", operation: "apply", resource: &v3beta1ac.DoguApplyConfiguration{}, prepare: discoverSource, message: "could not persist migration source PV"},
		{name: "get source PV", operation: "get", resource: &corev1.PersistentVolume{}, message: "could not get migration source PV"},
		{name: "retain source PV", operation: "update", resource: &corev1.PersistentVolume{}, message: "could not set PV reclaim policy to Retain", prepare: func(t *testing.T, f *pvcMigrationFixture) {
			f.pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimDelete
			require.NoError(t, f.client.Update(t.Context(), f.pv))
		}},
		{name: "list claims for deletion", operation: "list", resource: &corev1.PersistentVolumeClaimList{}, message: "could not list persistent volume claims"},
		{name: "delete source claim", operation: "delete", resource: &corev1.PersistentVolumeClaim{}, prepare: addSourceClaim, message: "could not delete dogu nexus pvcs"},
		{name: "reserve source PV", operation: "update", resource: &corev1.PersistentVolume{}, message: "could not reserve PV for migration PVC", prepare: func(t *testing.T, f *pvcMigrationFixture) {
			f.pv.Spec.ClaimRef = &corev1.ObjectReference{Name: "source", Namespace: "ecosystem"}
			require.NoError(t, f.client.Update(t.Context(), f.pv))
		}},
		{name: "get migration claim", operation: "get", resource: &corev1.PersistentVolumeClaim{}, message: "could not get migration PVC"},
		{name: "create migration claim", operation: "create", resource: &corev1.PersistentVolumeClaim{}, message: "could not create migration PVC", prepare: func(t *testing.T, f *pvcMigrationFixture) {
			require.NoError(t, f.client.Delete(t.Context(), f.claim))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPVCMigrationFixture(t)
			f.claim.Status.Phase = corev1.ClaimPending
			require.NoError(t, f.client.Status().Update(t.Context(), f.claim))
			if tt.prepare != nil {
				tt.prepare(t, f)
			}
			apiErr := errors.New("API request rejected")
			failed := false
			fail := func(operation string, obj any) error {
				require.False(t, failed, "no further API calls are allowed after a failure")
				if operation == tt.operation && reflect.TypeOf(obj) == reflect.TypeOf(tt.resource) {
					failed = true
					return apiErr
				}
				return nil
			}
			f.step.client = interceptor.NewClient(f.client, interceptor.Funcs{
				Apply: func(ctx context.Context, c client.WithWatch, configuration runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
					if err := fail("apply", configuration); err != nil {
						return err
					}
					return c.Apply(ctx, configuration, opts...)
				},
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if err := fail("get", obj); err != nil {
						return err
					}
					return c.Get(ctx, key, obj, opts...)
				},
				List: func(ctx context.Context, c client.WithWatch, obj client.ObjectList, opts ...client.ListOption) error {
					if err := fail("list", obj); err != nil {
						return err
					}
					return c.List(ctx, obj, opts...)
				},
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if err := fail("create", obj); err != nil {
						return err
					}
					return c.Create(ctx, obj, opts...)
				},
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if err := fail("update", obj); err != nil {
						return err
					}
					return c.Update(ctx, obj, opts...)
				},
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if err := fail("delete", obj); err != nil {
						return err
					}
					return c.Delete(ctx, obj, opts...)
				},
			})

			result := f.step.Run(t.Context(), f.dogu)
			require.True(t, failed, "expected API operation was not reached")
			require.ErrorIs(t, result.Err, apiErr)
			if tt.message != "" {
				assert.ErrorContains(t, result.Err, tt.message)
			}
			assert.False(t, result.Continue)
			assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
			assert.Equal(t, result.Err.Error(), result.ReadyMessage)
			assert.Empty(t, f.recorder.Events)
			stored := &v3beta1.Dogu{}
			require.NoError(t, f.client.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), stored))
			assert.Nil(t, meta.FindStatusCondition(stored.Status.Conditions, v3beta1.ConditionReady))
		})
	}
}
