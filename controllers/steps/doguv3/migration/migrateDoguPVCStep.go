package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrateDoguPVCStep struct {
	client        client.Client
	eventRecorder EventRecorder
}

const (
	requeueTime                 = time.Second
	migrationSourcePVAnnotation = "k8s.cloudogu.com/migration-source-pv"
)

func NewMigrateDoguPVCStep(k8sClient client.Client, recorder EventRecorder) *MigrateDoguPVCStep {
	return &MigrateDoguPVCStep{client: k8sClient, eventRecorder: recorder}
}

// Run Create a migration PVC for the dogu being migrated
func (s *MigrateDoguPVCStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if result := s.scaleDownV2Deployment(ctx, doguResource); !result.Continue {
		return result
	}
	pvName, result := s.persistSourcePVName(ctx, doguResource)
	if !result.Continue {
		return result
	}
	v2PV, result := s.retainSourcePV(ctx, pvName)
	if !result.Continue {
		return result
	}
	if result := s.deleteDoguPVCs(ctx, doguResource); !result.Continue {
		return result
	}
	if result := s.reservePVForMigration(ctx, doguResource, v2PV); !result.Continue {
		return result
	}
	if result := s.ensureMigrationPVCBound(ctx, doguResource, v2PV); !result.Continue {
		return result
	}
	if result := s.updateMigrationStatus(ctx, doguResource); !result.Continue {
		return result
	}

	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, "test")
	return stepsv3.Continue()
}

// scale down the dogu v2 deployment and reconcile until it is scaled down
func (s *MigrateDoguPVCStep) scaleDownV2Deployment(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	deployments := &appsv1.DeploymentList{}
	err := s.client.List(ctx, deployments, client.InNamespace(doguResource.Namespace), client.MatchingLabels{"dogu.name": doguResource.Spec.Name})
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("could not list v2 dogu deployment for %s: %w", doguResource.Spec.Name, err), v3beta1.ReasonMigrationPending)
	}
	if len(deployments.Items) != 1 {
		return stepsv3.RequeueWithError(fmt.Errorf("found %d v2 deployments for dogu %s, expected exactly one", len(deployments.Items), doguResource.Spec.Name), v3beta1.ReasonMigrationPending)
	}

	v2Deployment := &deployments.Items[0]
	if *v2Deployment.Spec.Replicas > 0 {
		v2Deployment.Spec.Replicas = new(int32(0))

		err = s.client.Update(ctx, v2Deployment)
		if err != nil {
			return stepsv3.RequeueWithError(fmt.Errorf("could not scale down v2 deployment: %w", err), v3beta1.ReasonMigrationPending)
		}
		return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for deployment to be scaled down")
	}

	return stepsv3.Continue()
}

// save the name of the source pv as an annotation on the v3 resource, otherwise the name gets lost when the pvc is deleted
func (s *MigrateDoguPVCStep) persistSourcePVName(ctx context.Context, doguResource *v3beta1.Dogu) (string, stepsv3.StepResult) {
	pvName := doguResource.Annotations[migrationSourcePVAnnotation]
	if pvName == "" {
		pvcs := &corev1.PersistentVolumeClaimList{}
		if err := s.client.List(ctx, pvcs, client.InNamespace(doguResource.Namespace), client.MatchingLabels{"dogu.name": doguResource.Spec.Name}); err != nil {
			return "", stepsv3.RequeueWithError(fmt.Errorf("could not list source PVCs for dogu %s: %w", doguResource.Spec.Name, err), v3beta1.ReasonMigrationPending)
		}
		if len(pvcs.Items) != 1 {
			return "", stepsv3.RequeueWithError(fmt.Errorf("found %d source PVCs for dogu %s, expected exactly one", len(pvcs.Items), doguResource.Spec.Name), v3beta1.ReasonMigrationPending)
		}
		pvName = pvcs.Items[0].Spec.VolumeName
		if pvName == "" {
			return "", stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for source PVC to be bound to a PV")
		}
		updatedDogu := doguResource.DeepCopy()
		if updatedDogu.Annotations == nil {
			updatedDogu.Annotations = make(map[string]string)
		}
		updatedDogu.Annotations[migrationSourcePVAnnotation] = pvName
		if err := s.client.Update(ctx, updatedDogu); err != nil {
			return "", stepsv3.RequeueWithError(fmt.Errorf("could not persist migration source PV: %w", err), v3beta1.ReasonMigrationPending)
		}
		*doguResource = *updatedDogu
		return "", stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "migration source PV persisted")
	}
	return pvName, stepsv3.Continue()
}

// set the reclaim policy on the pv to retain so the pv doesn't get deleted when the pvc is gone
func (s *MigrateDoguPVCStep) retainSourcePV(ctx context.Context, pvName string) (*corev1.PersistentVolume, stepsv3.StepResult) {
	v2PV := &corev1.PersistentVolume{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: pvName}, v2PV); err != nil {
		return nil, stepsv3.RequeueWithError(fmt.Errorf("could not get migration source PV %s: %w", pvName, err), v3beta1.ReasonMigrationPending)
	}

	if v2PV.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		v2PV.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		err := s.client.Update(ctx, v2PV)
		if err != nil {
			return nil, stepsv3.RequeueWithError(fmt.Errorf("could not set PV reclaim policy to Retain: %w", err), v3beta1.ReasonMigrationPending)
		}
		return nil, stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for PersistentVolumeReclaimPolicy to be updated")
	}
	return v2PV, stepsv3.Continue()
}

// delete all pvcs of the dogu, which should just be one
func (s *MigrateDoguPVCStep) deleteDoguPVCs(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := s.client.List(ctx, pvcs, client.InNamespace(doguResource.Namespace), client.MatchingLabels{"dogu.name": doguResource.Spec.Name})
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("could not list persistent volume claims for dogu %s: %w", doguResource.Spec.Name, err), v3beta1.ReasonMigrationPending)
	}

	if len(pvcs.Items) > 0 {
		for _, p := range pvcs.Items {
			err := s.client.Delete(ctx, &p)
			if err != nil {
				return stepsv3.RequeueWithError(fmt.Errorf("could not delete dogu %s pvcs: %w", doguResource.Spec.Name, err), v3beta1.ReasonMigrationPending)
			}
		}
		return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for pvc deletion")
	}

	return stepsv3.Continue()
}

// set the claimref on the pv
func (s *MigrateDoguPVCStep) reservePVForMigration(ctx context.Context, doguResource *v3beta1.Dogu, v2PV *corev1.PersistentVolume) stepsv3.StepResult {
	if v2PV.Spec.ClaimRef == nil || v2PV.Spec.ClaimRef.Name != doguResource.Spec.Name+"-migration" || v2PV.Spec.ClaimRef.Namespace != doguResource.Namespace {
		v2PV.Spec.ClaimRef = &corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
			Name:       doguResource.Spec.Name + "-migration",
			Namespace:  doguResource.Namespace,
		}
		err := s.client.Update(ctx, v2PV)
		if err != nil {
			return stepsv3.RequeueWithError(fmt.Errorf("could not reserve PV for migration PVC: %w", err), v3beta1.ReasonMigrationPending)
		}
		return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for PV reservation for migration PVC")
	}

	return stepsv3.Continue()
}

// create a migration pvc and ensure it is bound to the pv
func (s *MigrateDoguPVCStep) ensureMigrationPVCBound(ctx context.Context, doguResource *v3beta1.Dogu, v2PV *corev1.PersistentVolume) stepsv3.StepResult {
	migrationPVC := &corev1.PersistentVolumeClaim{}
	err := s.client.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name + "-migration"}, migrationPVC)
	if apierrors.IsNotFound(err) {
		newPVC := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      doguResource.Spec.Name + "-migration",
				Namespace: doguResource.Namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: v2PV.Spec.AccessModes,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: v2PV.Spec.Capacity[corev1.ResourceStorage],
					},
				},
				VolumeName:       v2PV.Name,
				StorageClassName: &v2PV.Spec.StorageClassName,
				VolumeMode:       v2PV.Spec.VolumeMode,
			},
		}

		err = s.client.Create(ctx, newPVC)
		if err != nil {
			return stepsv3.RequeueWithError(fmt.Errorf("could not create migration PVC: %w", err), v3beta1.ReasonMigrationPending)
		}
		return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for migration PVC to be bound to the source PV")
	} else if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("could not get migration PVC for dogu %s: %w", doguResource.Spec.Name, err), v3beta1.ReasonMigrationPending)
	}

	if migrationPVC.Status.Phase != corev1.ClaimBound || migrationPVC.UID == "" || v2PV.Spec.ClaimRef.UID != migrationPVC.UID {
		return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "wait for migration PVC to be bound to the source PV")
	}

	return stepsv3.Continue()
}

// update the dogu resource status
func (s *MigrateDoguPVCStep) updateMigrationStatus(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             v3beta1.ReasonMigrationPending,
		Message:            "PVC migration completed",
		ObservedGeneration: doguResource.Generation,
	}) {
		if err := s.client.Status().Update(ctx, doguResource); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonMigrationPending)
		}
	}

	return stepsv3.Continue()
}
