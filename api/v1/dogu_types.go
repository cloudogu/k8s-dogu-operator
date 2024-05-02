package v1

import (
	"context"
	"embed"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// This embed provides the crd for other applications. They can import this package and use the yaml file
// for the CRD in e.g. integration tests. Otherwise, this file would not be present in the golang vendor directory.
// The file gets refreshed by copying from controller-gen by the "crd-helm-generate/crd-copy-for-go-embedding" make target.
//
//go:embed k8s.cloudogu.com_dogus.yaml
var _ embed.FS

const (
	// RequeueTimeMultiplerForEachRequeue defines the factor to multiple the requeue time of a failed dogu crd operation
	RequeueTimeMultiplerForEachRequeue = 2
	// RequeueTimeInitialRequeueTime defines the initial value of the requeue time
	RequeueTimeInitialRequeueTime = time.Second * 5
	// RequeueTimeMaxRequeueTime defines the maximum amount of time to wait for a requeue of a dogu resource
	RequeueTimeMaxRequeueTime = time.Hour * 6
	// DefaultVolumeSize is the default size of a new dogu volume if no volume size is specified in the dogu resource.
	DefaultVolumeSize = "2Gi"
)

const (
	// DoguLabelName is used to select a dogu pod by name.
	DoguLabelName = "dogu.name"
	// DoguLabelVersion is used to select a dogu pod by version.
	DoguLabelVersion = "dogu.version"
)

// DoguSpec defines the desired state of a Dogu
type DoguSpec struct {
	// Name of the dogu (e.g. official/ldap)
	Name string `json:"name,omitempty"`
	// Version of the dogu (e.g. 2.4.48-3)
	Version string `json:"version,omitempty"`
	// Resources of the dogu (e.g. dataVolumeSize)
	Resources DoguResources `json:"resources,omitempty"`
	// SupportMode indicates whether the dogu should be restarted in the support mode (f. e. to recover manually from
	// a crash loop).
	SupportMode bool `json:"supportMode,omitempty"`
	// Stopped indicates whether the dogu should be running (stopped=false) or not (stopped=true).
	Stopped bool `json:"stopped,omitempty"`
	// UpgradeConfig contains options to manipulate the upgrade process.
	UpgradeConfig UpgradeConfig `json:"upgradeConfig,omitempty"`
	// AdditionalIngressAnnotations provides additional annotations that get included into the dogu's ingress rules.
	AdditionalIngressAnnotations IngressAnnotations `json:"additionalIngressAnnotations,omitempty"`
}

// IngressAnnotations are annotations of nginx-ingress rules.
type IngressAnnotations map[string]string

// UpgradeConfig contains configuration hints for the dogu operator regarding aspects during the upgrade of dogus.
type UpgradeConfig struct {
	// AllowNamespaceSwitch lets a dogu switch its dogu namespace during an upgrade. The dogu must be technically the
	// same dogu which did reside in a different namespace. The remote dogu's version must be equal to or greater than
	// the version of the local dogu.
	AllowNamespaceSwitch bool `json:"allowNamespaceSwitch,omitempty"`
	// ForceUpgrade allows to install the same or even lower dogu version than already is installed. Please note, that
	// possible data loss may occur by inappropriate dogu downgrading.
	ForceUpgrade bool `json:"forceUpgrade,omitempty"`
}

// DoguResources defines the physical resources used by the dogu.
type DoguResources struct {
	// dataVolumeSize represents the current size of the volume. Increasing this value leads to an automatic volume
	// expansion. This includes a downtime for the respective dogu. The default size for volumes is "2Gi".
	// It is not possible to lower the volume size after an expansion. This will introduce an inconsistent state for the
	// dogu.
	DataVolumeSize string `json:"dataVolumeSize,omitempty"`
}

type HealthStatus string

const (
	PendingHealthStatus     HealthStatus = ""
	AvailableHealthStatus   HealthStatus = "available"
	UnavailableHealthStatus HealthStatus = "unavailable"
	UnknownHealthStatus     HealthStatus = "unknown"
)

// DoguStatus defines the observed state of a Dogu.
type DoguStatus struct {
	// Status represents the state of the Dogu in the ecosystem
	Status string `json:"status"`
	// RequeueTime contains time necessary to perform the next requeue
	RequeueTime time.Duration `json:"requeueTime"`
	// RequeuePhase is the actual phase of the dogu resource used for a currently running async process.
	RequeuePhase string `json:"requeuePhase"`
	// Health describes the health status of the dogu
	Health HealthStatus `json:"health,omitempty"`
	// InstalledVersion of the dogu (e.g. 2.4.48-3)
	InstalledVersion string `json:"installedVersion,omitempty"`
	// Stopped shows if the dogu has been stopped or not.
	Stopped bool `json:"stopped,omitempty"`
}

func (d *Dogu) NextRequeueWithRetry(ctx context.Context, client client.Client) (time.Duration, error) {
	var requeueTime time.Duration
	err := retry.OnConflict(func() error {
		fetchErr := d.refreshDoguValue(ctx, client)
		if fetchErr != nil {
			return fetchErr
		}
		requeueTime = d.Status.NextRequeue()

		return d.Update(ctx, client)
	})

	if err != nil {
		return 0, err
	}

	return requeueTime, err
}

// NextRequeue increases the requeue time of the dogu status and returns the new requeue time
func (ds *DoguStatus) NextRequeue() time.Duration {
	if ds.RequeueTime == 0 {
		ds.ResetRequeueTime()
	}

	newRequeueTime := ds.RequeueTime * RequeueTimeMultiplerForEachRequeue
	if newRequeueTime >= RequeueTimeMaxRequeueTime {
		ds.RequeueTime = RequeueTimeMaxRequeueTime
	} else {
		ds.RequeueTime = newRequeueTime
	}
	return ds.RequeueTime
}

// ResetRequeueTime resets the requeue timer to the initial value
func (ds *DoguStatus) ResetRequeueTime() {
	ds.RequeueTime = RequeueTimeInitialRequeueTime
}

const (
	DoguStatusNotInstalled = ""
	DoguStatusInstalling   = "installing"
	DoguStatusUpgrading    = "upgrading"
	DoguStatusDeleting     = "deleting"
	DoguStatusInstalled    = "installed"
	DoguStatusPVCResizing  = "resizing PVC"
	DoguStatusStarting     = "starting"
	DoguStatusStopping     = "stopping"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Dogu is the Schema for the dogus API
type Dogu struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DoguSpec   `json:"spec,omitempty"`
	Status DoguStatus `json:"status,omitempty"`
}

// GetDataVolumeName returns the data volume name for the dogu resource for volumes with backup
func (d *Dogu) GetDataVolumeName() string {
	return d.Name + "-data"
}

// GetEphemeralDataVolumeName returns the data volume name for the dogu resource for volumes without backup
func (d *Dogu) GetEphemeralDataVolumeName() string {
	return d.Name + "-ephemeral"
}

// GetPrivateKeySecretName returns the name of the dogus secret resource.
func (d *Dogu) GetPrivateKeySecretName() string {
	return d.Name + "-private"
}

// GetObjectKey returns the object key with the actual name and namespace from the dogu resource
func (d *Dogu) GetObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
}

// GetDevelopmentDoguMapKey returns the object key for the custom dogu descriptor with the actual name and namespace
// from the dogu resource.
func (d *Dogu) GetDevelopmentDoguMapKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name + "-descriptor",
	}
}

// GetSecretObjectKey returns the object key for the config map containing values that should be encrypted for the dogu
func (d *Dogu) GetSecretObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name + "-secrets",
	}
}

// GetPrivateKeyObjectKey returns the object key for the secret containing the private key for the dogu.
func (d *Dogu) GetPrivateKeyObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Name:      d.GetPrivateKeySecretName(),
		Namespace: d.Namespace,
	}
}

// GetObjectMeta return the object meta with the actual name and namespace from the dogu resource
func (d *Dogu) GetObjectMeta() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
}

// Update updates the dogu's status property in the cluster state.
func (d *Dogu) Update(ctx context.Context, client client.Client) error {
	updateError := client.Status().Update(ctx, d)
	if updateError != nil {
		return fmt.Errorf("failed to update dogu status: %w", updateError)
	}

	return nil
}

// changeRequeuePhase changes the requeue phase of this dogu resource and applies it to the cluster state.
func (d *Dogu) changeRequeuePhase(ctx context.Context, client client.Client, phase string) error {
	d.Status.RequeuePhase = phase
	return d.Update(ctx, client)
}

// ChangeRequeuePhaseWithRetry refreshes the dogu resource and tries to set the requeue phase.
// If a conflict error occurs this method will retry the operation.
func (d *Dogu) ChangeRequeuePhaseWithRetry(ctx context.Context, client client.Client, phase string) error {
	return retry.OnConflict(func() error {
		err := d.refreshDoguValue(ctx, client)
		if err != nil {
			return err
		}

		return d.changeRequeuePhase(ctx, client, phase)
	})
}

func (d *Dogu) refreshDoguValue(ctx context.Context, client client.Client) error {
	dogu := &Dogu{}
	err := client.Get(ctx, d.GetObjectKey(), dogu)
	if err != nil {
		return err
	}
	*d = *dogu

	return nil
}

// changeState changes the state of this dogu resource and applies it to the cluster state.
func (d *Dogu) changeState(ctx context.Context, client client.Client, newStatus string) error {
	d.Status.Status = newStatus
	return d.Update(ctx, client)
}

// ChangeStateWithRetry refreshes the dogu resource and tries to set the state.
// If a conflict error occurs this method will retry the operation.
func (d *Dogu) ChangeStateWithRetry(ctx context.Context, client client.Client, newStatus string) error {
	return retry.OnConflict(func() error {
		err := d.refreshDoguValue(ctx, client)
		if err != nil {
			return err
		}

		return d.changeState(ctx, client, newStatus)
	})
}

// GetPodLabels returns labels that select a pod being associated with this dogu.
func (d *Dogu) GetPodLabels() CesMatchingLabels {
	return map[string]string{
		DoguLabelName:    d.Name,
		DoguLabelVersion: d.Spec.Version,
	}
}

// GetDoguNameLabel returns labels that select any resource being associated with this dogu.
func (d *Dogu) GetDoguNameLabel() CesMatchingLabels {
	return map[string]string{
		DoguLabelName: d.Name,
	}
}

// GetPod returns a pod for this dogu. An error is returned if either no pod or more than one pod is found.
func (d *Dogu) GetPod(ctx context.Context, cli client.Client) (*corev1.Pod, error) {
	labels := d.GetPodLabels()
	return GetPodForLabels(ctx, cli, labels)
}

// GetDataPVC returns the data pvc for this dogu.
func (d *Dogu) GetDataPVC(ctx context.Context, cli client.Client) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := cli.Get(ctx, d.GetObjectKey(), pvc)
	if err != nil {
		return nil, fmt.Errorf("failed to get data pvc for dogu %s: %w", d.Name, err)
	}

	return pvc, nil
}

// GetDeployment returns the deployment for this dogu.
func (d *Dogu) GetDeployment(ctx context.Context, cli client.Client) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	err := cli.Get(ctx, d.GetObjectKey(), deploy)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment for dogu %s: %w", d.Name, err)
	}

	return deploy, nil
}

// GetDataVolumeSize returns the dataVolumeSize of the dogu. If no size is set the default size will be returned.
func (d *Dogu) GetDataVolumeSize() resource.Quantity {
	doguTargetDataVolumeSize := resource.MustParse(DefaultVolumeSize)
	if d.Spec.Resources.DataVolumeSize != "" {
		doguTargetDataVolumeSize = resource.MustParse(d.Spec.Resources.DataVolumeSize)
	}

	return doguTargetDataVolumeSize
}

// GetPrivateKeySecret returns the private key secret for this dogu.
func (d *Dogu) GetPrivateKeySecret(ctx context.Context, cli client.Client) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := cli.Get(ctx, d.GetPrivateKeyObjectKey(), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key secret for dogu %s: %w", d.Name, err)
	}

	return secret, nil
}

// +kubebuilder:object:root=true

// DoguList contains a list of Dogu
type DoguList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dogu `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dogu{}, &DoguList{})
}

// DevelopmentDoguMap is a config map that is especially used to when developing a dogu. The map contains a custom
// dogu.json in the data filed with the "dogu.json" identifier.
type DevelopmentDoguMap corev1.ConfigMap

// DeleteFromCluster deletes this development config map from the cluster.
func (ddm *DevelopmentDoguMap) DeleteFromCluster(ctx context.Context, client client.Client) error {
	err := client.Delete(ctx, ddm.ToConfigMap())
	if err != nil {
		return fmt.Errorf("failed to delete custom dogu development map %s: %w", ddm.Name, err)
	}

	return nil
}

// ToConfigMap returns the development dogu map as config map pointer.
func (ddm *DevelopmentDoguMap) ToConfigMap() *corev1.ConfigMap {
	configMap := corev1.ConfigMap(*ddm)
	return &configMap
}

// CesMatchingLabels provides a convenient way to handle multiple labels for resource selection.
type CesMatchingLabels client.MatchingLabels

// Add takes the currently existing labels from this object and returns a sum of all provided labels as a new object.
func (cml CesMatchingLabels) Add(moreLabels CesMatchingLabels) CesMatchingLabels {
	result := CesMatchingLabels{}
	for key, value := range cml {
		result[key] = value
	}

	for key, value := range moreLabels {
		result[key] = value
	}

	return result
}
