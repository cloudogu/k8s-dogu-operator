/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"
	"time"

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

// DoguSpec defines the desired state of a Dogu
type DoguSpec struct {
	// Name of the dogu (e.g. official/ldap)
	Name string `json:"name,omitempty"`
	// Version of the dogu (e.g. 2.4.48-3)
	Version string `json:"version,omitempty"`
	//
	UpgradeConfig UpgradeConfig `json:"upgradeConfig"`
}

// UpgradeConfig contains configuration hints for the dogu operator regarding aspects during the upgrade of dogus.
type UpgradeConfig struct {
	// AllowNamespaceSwitch lets a dogu switch its dogu namespace during an upgrade. The dogu must be technically the
	// same dogu which did reside in a different namespace. The remote dogu's version must be equal to or greater than
	// the version of the local dogu.
	AllowNamespaceSwitch bool `json:"allowNamespaceSwitch"`
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
func (d Dogu) GetDataVolumeName() string {
	return d.Name + "-data"
}

// GetPrivateVolumeName returns the private volume name for the dogu resource
func (d Dogu) GetPrivateVolumeName() string {
	return d.Name + "-private"
}

// GetObjectKey returns the object key with the actual name and namespace from the dogu resource
func (d Dogu) GetObjectKey() *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
}

// GetDescriptorObjectKey returns the object key for the custom dogu descriptor with the actual name and namespace from
// the dogu resource
func (d Dogu) GetDescriptorObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name + "-descriptor",
	}
}

// GetSecretObjectKey returns the object key for the config map containing values that should be encrypted for the dogu
func (d Dogu) GetSecretObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name + "-secrets",
	}
}

// GetObjectMeta return the object meta with the actual name and namespace from the dogu resource
func (d Dogu) GetObjectMeta() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
}

// Update removes all messages from the message log
func (d *Dogu) Update(ctx context.Context, client client.Client) error {
	updateError := client.Status().Update(ctx, d)
	if updateError != nil {
		return fmt.Errorf("failed to update dogu status: %w", updateError)
	}

	return nil
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
