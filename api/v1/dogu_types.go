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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DoguSpec defines the desired state of Dogu
type DoguSpec struct {
	// Name of the dogu (e.g. official/ldap)
	Name string `json:"name,omitempty"`
	// Version of the dogu (e.g. 2.4.48-3)
	Version string `json:"version,omitempty"`
}

// DoguStatus defines the observed state of Dogu
type DoguStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

// GetDescriptorObjectKey returns the object key for the custom dogu descriptor with the actual name and namespace from the dogu resource
func (d Dogu) GetDescriptorObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: d.Namespace,
		Name:      d.Name + "-descriptor",
	}
}

// GetObjectMeta return the object meta with the actual name and namespace from the dogu resource
func (d Dogu) GetObjectMeta() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Namespace: d.Namespace,
		Name:      d.Name,
	}
}

//+kubebuilder:object:root=true

// DoguList contains a list of Dogu
type DoguList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dogu `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dogu{}, &DoguList{})
}
