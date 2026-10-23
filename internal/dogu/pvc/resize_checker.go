package pvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizeRequest describes a required PVC expansion.
type ResizeRequest struct {
	PVC     client.ObjectKey
	Desired resource.Quantity
}

// CheckResult contains only PVCs that require an expansion or whose expansion to the desired size is still running.
type CheckResult struct {
	ResizeRequests []ResizeRequest
	InProgress     []client.ObjectKey
	// OutdatedStatefulSets are the UIDs of live StatefulSets whose immutable claim templates request another size than rendered.
	OutdatedStatefulSets []types.UID
}

type pvcOutcome int

const (
	pvcUnchanged pvcOutcome = iota
	pvcNeedsResize
	pvcResizeInProgress
)

func (r *CheckResult) add(request ResizeRequest, outcome pvcOutcome) {
	switch outcome {
	case pvcNeedsResize:
		r.ResizeRequests = append(r.ResizeRequests, request)
	case pvcResizeInProgress:
		r.InProgress = append(r.InProgress, request.PVC)
	case pvcUnchanged:
	}
}

// ResizeChecker checks rendered Kubernetes objects against live PVCs without changing cluster resources.
type ResizeChecker interface {
	Check(ctx context.Context, targetNamespace string, renderedObjects []*unstructured.Unstructured) (CheckResult, error)
}

type resizeChecker struct {
	k8s client.Reader
}

// NewResizeChecker creates a checker that reads live PVCs with the given Kubernetes client.
func NewResizeChecker(k8s client.Reader) ResizeChecker {
	return &resizeChecker{k8s: k8s}
}

type desiredClaim struct {
	key          client.ObjectKey
	storage      resource.Quantity
	storageClass *string
}

type desiredClaimTemplate struct {
	statefulSet  client.ObjectKey
	name         string
	storage      resource.Quantity
	storageClass *string
}

func (c *resizeChecker) Check(ctx context.Context, targetNamespace string, renderedObjects []*unstructured.Unstructured) (CheckResult, error) {
	if targetNamespace == "" {
		return CheckResult{}, errors.New("target namespace must not be empty")
	}

	desiredClaims, desiredTemplates, extractionErr := extractDesiredClaims(renderedObjects, targetNamespace)
	result := CheckResult{}
	checkErr := extractionErr

	for _, desired := range desiredClaims {
		request, outcome, err := c.checkPVC(ctx, desired)
		if err != nil {
			checkErr = errors.Join(checkErr, err)
			continue
		}
		result.add(request, outcome)
	}

	checkErr = errors.Join(checkErr, c.checkStatefulSetClaims(ctx, desiredTemplates, &result))

	outdated, err := c.outdatedStatefulSets(ctx, desiredTemplates)
	checkErr = errors.Join(checkErr, err)
	result.OutdatedStatefulSets = outdated

	if checkErr != nil {
		return CheckResult{}, checkErr
	}

	return result, nil
}

func (c *resizeChecker) checkStatefulSetClaims(ctx context.Context, desiredTemplates []desiredClaimTemplate, result *CheckResult) error {
	templatesByNamespace := map[string][]desiredClaimTemplate{}
	for _, template := range desiredTemplates {
		templatesByNamespace[template.statefulSet.Namespace] = append(templatesByNamespace[template.statefulSet.Namespace], template)
	}

	var checkErr error
	for namespace, templates := range templatesByNamespace {
		livePVCs := &corev1.PersistentVolumeClaimList{}
		if err := c.k8s.List(ctx, livePVCs, client.InNamespace(namespace)); err != nil {
			checkErr = errors.Join(checkErr, fmt.Errorf("failed to list live PVCs in namespace %q: %w", namespace, err))
			continue
		}

		for i := range livePVCs.Items {
			livePVC := &livePVCs.Items[i]
			for _, template := range templates {
				if !MatchesStatefulSetClaim(livePVC.Name, template.name, template.statefulSet.Name) {
					continue
				}

				desired := desiredClaim{
					key:          client.ObjectKeyFromObject(livePVC),
					storage:      template.storage.DeepCopy(),
					storageClass: copyStringPointer(template.storageClass),
				}
				request, outcome, err := checkLivePVC(desired, livePVC)
				if err != nil {
					checkErr = errors.Join(checkErr, err)
					continue
				}
				result.add(request, outcome)
			}
		}
	}

	return checkErr
}

// outdatedStatefulSets finds live StatefulSets a Helm upgrade cannot change because their claim template sizes differ.
func (c *resizeChecker) outdatedStatefulSets(ctx context.Context, desiredTemplates []desiredClaimTemplate) ([]types.UID, error) {
	var outdated []types.UID
	var checkErr error
	for _, template := range desiredTemplates {
		live := &appsv1.StatefulSet{}
		err := c.k8s.Get(ctx, template.statefulSet, live)
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			checkErr = errors.Join(checkErr, fmt.Errorf("failed to read live StatefulSet %q: %w", template.statefulSet, err))
			continue
		}
		if slices.Contains(outdated, live.UID) {
			continue
		}

		for _, liveTemplate := range live.Spec.VolumeClaimTemplates {
			liveStorage, found := liveTemplate.Spec.Resources.Requests[corev1.ResourceStorage]
			if liveTemplate.Name == template.name && found && liveStorage.Cmp(template.storage) != 0 {
				outdated = append(outdated, live.UID)
				break
			}
		}
	}

	return outdated, checkErr
}

func (c *resizeChecker) checkPVC(ctx context.Context, desired desiredClaim) (ResizeRequest, pvcOutcome, error) {
	livePVC := &corev1.PersistentVolumeClaim{}
	err := c.k8s.Get(ctx, desired.key, livePVC)
	if apierrors.IsNotFound(err) {
		return ResizeRequest{}, pvcUnchanged, nil
	}
	if err != nil {
		return ResizeRequest{}, pvcUnchanged, fmt.Errorf("failed to read live PVC %q: %w", desired.key, err)
	}

	return checkLivePVC(desired, livePVC)
}

func checkLivePVC(desired desiredClaim, livePVC *corev1.PersistentVolumeClaim) (ResizeRequest, pvcOutcome, error) {
	request := ResizeRequest{
		PVC:     desired.key,
		Desired: desired.storage.DeepCopy(),
	}

	var validationErr error
	if desired.storageClass != nil && !ptr.Equal(desired.storageClass, livePVC.Spec.StorageClassName) {
		validationErr = &StorageClassImmutableError{
			PVC:     desired.key,
			Current: copyStringPointer(livePVC.Spec.StorageClassName),
			Desired: copyStringPointer(desired.storageClass),
		}
	}

	if validationErr == nil && resizeInProgress(desired.storage, livePVC) {
		return request, pvcResizeInProgress, nil
	}

	currentRequest, err := currentStorageRequest(livePVC)
	if err != nil {
		return ResizeRequest{}, pvcUnchanged, errors.Join(validationErr, err)
	}

	if desired.storage.Cmp(currentRequest) < 0 {
		validationErr = errors.Join(validationErr, &VolumeShrinkError{
			PVC:     desired.key,
			Current: currentRequest.DeepCopy(),
			Desired: desired.storage.DeepCopy(),
		})
	}
	if validationErr != nil {
		return ResizeRequest{}, pvcUnchanged, validationErr
	}
	if desired.storage.Cmp(currentRequest) > 0 {
		return request, pvcNeedsResize, nil
	}

	return ResizeRequest{}, pvcUnchanged, nil
}

// resizeInProgress reports whether the bound PVC already requests the desired size and its expansion has not finished without error.
func resizeInProgress(desired resource.Quantity, pvc *corev1.PersistentVolumeClaim) bool {
	requested, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found || requested.Cmp(desired) != 0 || pvc.Status.Phase != corev1.ClaimBound ||
		hasTrueCondition(pvc, corev1.PersistentVolumeClaimControllerResizeError, corev1.PersistentVolumeClaimNodeResizeError) {
		return false
	}

	capacity, found := pvc.Status.Capacity[corev1.ResourceStorage]
	return found && capacity.Cmp(requested) < 0 ||
		pvc.Status.AllocatedResourceStatuses[corev1.ResourceStorage] != "" ||
		hasTrueCondition(pvc, corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimFileSystemResizePending)
}

func hasTrueCondition(pvc *corev1.PersistentVolumeClaim, conditionTypes ...corev1.PersistentVolumeClaimConditionType) bool {
	for _, condition := range pvc.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && slices.Contains(conditionTypes, condition.Type) {
			return true
		}
	}

	return false
}

func currentStorageRequest(pvc *corev1.PersistentVolumeClaim) (resource.Quantity, error) {
	key := client.ObjectKeyFromObject(pvc)
	if pvc.Status.Phase != corev1.ClaimBound {
		return resource.Quantity{}, fmt.Errorf("live PVC %q is not bound: phase %q", key, pvc.Status.Phase)
	}

	if resizeStatus := pvc.Status.AllocatedResourceStatuses[corev1.ResourceStorage]; resizeStatus != "" {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has storage resize status %q", key, resizeStatus)
	}

	for _, condition := range pvc.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && isResizeCondition(condition.Type) {
			return resource.Quantity{}, fmt.Errorf("live PVC %q has active resize condition %q", key, condition.Type)
		}
	}

	requested, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has no storage request", key)
	}
	capacity, found := pvc.Status.Capacity[corev1.ResourceStorage]
	if !found {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has no storage capacity", key)
	}
	if requested.Cmp(capacity) > 0 {
		return resource.Quantity{}, fmt.Errorf("live PVC %q is not ready: storage request %s exceeds capacity %s", key, requested.String(), capacity.String())
	}

	return requested, nil
}

func isResizeCondition(conditionType corev1.PersistentVolumeClaimConditionType) bool {
	switch conditionType {
	case corev1.PersistentVolumeClaimResizing,
		corev1.PersistentVolumeClaimFileSystemResizePending,
		corev1.PersistentVolumeClaimControllerResizeError,
		corev1.PersistentVolumeClaimNodeResizeError:
		return true
	default:
		return false
	}
}

func extractDesiredClaims(renderedObjects []*unstructured.Unstructured, targetNamespace string) ([]desiredClaim, []desiredClaimTemplate, error) {
	var claims []desiredClaim
	var templates []desiredClaimTemplate
	var extractionErr error

	for _, unstructuredObject := range renderedObjects {
		switch unstructuredObject.GroupVersionKind().GroupKind() {
		case schema.GroupKind{Group: corev1.GroupName, Kind: "PersistentVolumeClaim"}:
			pvc := &corev1.PersistentVolumeClaim{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, pvc); err != nil {
				extractionErr = errors.Join(extractionErr, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err))
				continue
			}
			if pvc.Namespace == "" {
				pvc.Namespace = targetNamespace
			}

			claim, err := desiredClaimFromPVC(pvc)
			if err != nil {
				extractionErr = errors.Join(extractionErr, err)
				continue
			}
			claims = append(claims, claim)
		case schema.GroupKind{Group: appsv1.GroupName, Kind: "StatefulSet"}:
			statefulSet := &appsv1.StatefulSet{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, statefulSet); err != nil {
				extractionErr = errors.Join(extractionErr, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err))
				continue
			}
			if statefulSet.Namespace == "" {
				statefulSet.Namespace = targetNamespace
			}

			statefulSetTemplates, err := desiredClaimTemplatesFromStatefulSet(statefulSet)
			if err != nil {
				extractionErr = errors.Join(extractionErr, err)
			}
			templates = append(templates, statefulSetTemplates...)
		}
	}

	return claims, templates, extractionErr
}

func desiredClaimTemplatesFromStatefulSet(statefulSet *appsv1.StatefulSet) ([]desiredClaimTemplate, error) {
	templates := make([]desiredClaimTemplate, 0, len(statefulSet.Spec.VolumeClaimTemplates))
	var extractionErr error
	for _, template := range statefulSet.Spec.VolumeClaimTemplates {
		pvc := template.DeepCopy()
		pvc.Namespace = statefulSet.Namespace

		claim, err := desiredClaimFromPVC(pvc)
		if err != nil {
			extractionErr = errors.Join(extractionErr, err)
			continue
		}
		templates = append(templates, desiredClaimTemplate{
			statefulSet:  client.ObjectKeyFromObject(statefulSet),
			name:         template.Name,
			storage:      claim.storage,
			storageClass: claim.storageClass,
		})
	}

	return templates, extractionErr
}

// MatchesStatefulSetClaim reports whether the PVC was created from the StatefulSet's volume claim template.
func MatchesStatefulSetClaim(pvcName, templateName, statefulSetName string) bool {
	ordinal, found := strings.CutPrefix(pvcName, templateName+"-"+statefulSetName+"-")
	if !found || ordinal == "" || ordinal[0] == '0' && len(ordinal) > 1 {
		return false
	}

	for _, character := range ordinal {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}

func desiredClaimFromPVC(pvc *corev1.PersistentVolumeClaim) (desiredClaim, error) {
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == "" {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q uses unsupported static provisioning", client.ObjectKeyFromObject(pvc))
	}

	storage, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q has no storage request", client.ObjectKeyFromObject(pvc))
	}

	return desiredClaim{
		key:          client.ObjectKeyFromObject(pvc),
		storage:      storage.DeepCopy(),
		storageClass: copyStringPointer(pvc.Spec.StorageClassName),
	}, nil
}

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	return ptr.To(*value)
}
