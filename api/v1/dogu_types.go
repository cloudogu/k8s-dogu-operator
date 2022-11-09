package v1

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// RequeueTimeMultiplerForEachRequeue defines the factor to multiple the requeue time of a failed dogu crd operation
	RequeueTimeMultiplerForEachRequeue = 2
	// RequeueTimeInitialRequeueTime defines the initial value of the requeue time
	RequeueTimeInitialRequeueTime = time.Second * 5
	// RequeueTimeMaxRequeueTime defines the maximum amount of time to wait for a requeue of a dogu resource
	RequeueTimeMaxRequeueTime = time.Hour * 6
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
	// SupportMode indicates whether the dogu should be restarted in the support mode (f. e. to recover manually from
	// a crash loop).
	SupportMode bool `json:"supportMode,omitempty"`
	// UpgradeConfig contains options to manipulate the upgrade process.
	UpgradeConfig UpgradeConfig `json:"upgradeConfig,omitempty"`
}

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

// DoguStatus defines the observed state of a Dogu
type DoguStatus struct {
	// Status represents the state of the Dogu in the ecosystem
	Status string `json:"status"`
	// StatusMessages contains a list of status messages
	StatusMessages []string `json:"statusMessages"`
	// RequeueTime contains time necessary to perform the next requeue
	RequeueTime time.Duration `json:"requeueTime"`
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

// AddMessage adds a new entry to the message slice
func (ds *DoguStatus) AddMessage(message string) {
	if ds.StatusMessages == nil {
		ds.StatusMessages = []string{}
	}

	ds.StatusMessages = append(ds.StatusMessages, message)
}

// ClearMessages removes all messages from the message log
func (ds *DoguStatus) ClearMessages() {
	ds.StatusMessages = []string{}
}

const (
	DoguStatusNotInstalled = ""
	DoguStatusInstalling   = "installing"
	DoguStatusUpgrading    = "upgrading"
	DoguStatusDeleting     = "deleting"
	DoguStatusInstalled    = "installed"
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

// GetDataVolumeName returns the data volume name for the dogu resource
func (d *Dogu) GetDataVolumeName() string {
	return d.Name + "-data"
}

// GetPrivateVolumeName returns the private volume name for the dogu resource
func (d *Dogu) GetPrivateVolumeName() string {
	return d.Name + "-private"
}

// GetReservedVolumeName returns the reserved (e.g. for upgrades) volume name for the dogu resource.
func (d *Dogu) GetReservedVolumeName() string {
	return d.Name + "-reserved"
}

// GetReservedPVCName returns the reserved (e.g. for upgrades) PVC name for the dogu resource.
func (d *Dogu) GetReservedPVCName() string {
	return d.Name + "-reserved"
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

// ChangeState changes the state of this dogu resource and applies it to the cluster state.
func (d *Dogu) ChangeState(ctx context.Context, client client.Client, newStatus string) error {
	d.Status.Status = newStatus
	return d.Update(ctx, client)
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
func (d *Dogu) GetPod(ctx context.Context, cli client.Client) (*v1.Pod, error) {
	labels := d.GetPodLabels()
	return GetPodForLabels(ctx, cli, labels)
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
type DevelopmentDoguMap v1.ConfigMap

// DeleteFromCluster deletes this development config map from the cluster.
func (ddm *DevelopmentDoguMap) DeleteFromCluster(ctx context.Context, client client.Client) error {
	err := client.Delete(ctx, ddm.ToConfigMap())
	if err != nil {
		return fmt.Errorf("failed to delete custom dogu development map %s: %w", ddm.Name, err)
	}

	return nil
}

// ToConfigMap returns the development dogu map as config map pointer.
func (ddm *DevelopmentDoguMap) ToConfigMap() *v1.ConfigMap {
	configMap := v1.ConfigMap(*ddm)
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
