/*
This file was generated with "make generate-deepcopy".
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DoguRestartSpec defines the desired state of DoguRestart
type DoguRestartSpec struct {
	// DoguName references the dogu that should get restarted.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Dogu name is immutable"
	DoguName string `json:"doguName"`
}

// DoguRestartStatus defines the observed state of DoguRestart
type DoguRestartStatus struct {
	// Phase tracks the state of the restart process.
	Phase RestartStatusPhase `json:"phase,omitempty"`
}

type RestartStatusPhase string

func (rsp RestartStatusPhase) IsFailed() bool {
	return rsp != RestartStatusPhaseNew && !rsp.isInProgress() && !rsp.isSuccessful()
}

func (rsp RestartStatusPhase) isInProgress() bool {
	return rsp == RestartStatusPhaseStarting || rsp == RestartStatusPhaseStopping || rsp == RestartStatusPhaseStopped
}

func (rsp RestartStatusPhase) isSuccessful() bool {
	return rsp == RestartStatusPhaseCompleted
}

const (
	RestartStatusPhaseNew           RestartStatusPhase = ""
	RestartStatusPhaseStopping      RestartStatusPhase = "stopping"
	RestartStatusPhaseStopped       RestartStatusPhase = "stopped"
	RestartStatusPhaseStarting      RestartStatusPhase = "starting"
	RestartStatusPhaseCompleted     RestartStatusPhase = "completed"
	RestartStatusPhaseDoguNotFound  RestartStatusPhase = "dogu not found"
	RestartStatusPhaseFailedGetDogu RestartStatusPhase = "failed getting dogu"
	RestartStatusPhaseFailedStop    RestartStatusPhase = "stop failed"
	RestartStatusPhaseFailedStart   RestartStatusPhase = "start failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DoguRestart is the Schema for the dogurestarts API
type DoguRestart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DoguRestartSpec   `json:"spec,omitempty"`
	Status DoguRestartStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DoguRestartList contains a list of DoguRestart
type DoguRestartList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DoguRestart `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DoguRestart{}, &DoguRestartList{})
}
