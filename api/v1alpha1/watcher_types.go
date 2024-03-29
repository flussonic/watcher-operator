/*
Copyright 2024.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WatcherSpec defines the desired state of Watcher
type WatcherSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Docker image for watcher
	Image string `json:"image"`

	// (Optional) PodEnvVariables is a slice of environment variables that are added to the pods
	// Default: (empty list)
	// +optional
	PodEnvVariables []corev1.EnvVar `json:"env,omitempty"`

	// Database URL. You should launch it yourself. Only Postgres is supported
	Database string `json:"database"`

	// Central URL. You should launch it yourself
	Central string `json:"central,omitempty"`

	// Count of web workers
	WebWorkers int32 `json:"webWorkers,omitempty"`

	// Count of background job workers
	JobWorkers int32 `json:"jobWorkers,omitempty"`

	// (Optional) node selector for placing pods with Web worker instances.
	WebNodeSelector map[string]string `json:"webNodeSelector,omitempty" protobuf:"bytes,7,rep,name=webNodeSelector"`

	// (Optional) node selector for placing pods with Web worker instances.
	JobNodeSelector map[string]string `json:"jobNodeSelector,omitempty" protobuf:"bytes,7,rep,name=jobNodeSelector"`
}

// WatcherStatus defines the observed state of Watcher
type WatcherStatus struct {
	// metav1.TypeMeta `json:",inline"`
	// // Standard object's metadata.
	// // More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// // +optional
	// metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// // List of status conditions to indicate the status of certificates.
	// // Known condition types are `Ready` and `Issuing`.
	// Conditions []WatcherCondition `json:"conditions"`
}

// // WatcherCondition contains condition information about whole VMS cluster
// type WatcherCondition struct {
// 	// Type of whole cluster condition
// 	Type WatcherConditionType `json:"type"`

// 	// Status of the condition, one of True, False, Unknown.
// 	Status corev1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`

// 	// LastTransitionTime is the timestamp corresponding to the last status
// 	// change of this condition.
// 	LastTransitionTime metav1.Time	`json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`

// 	// (brief) reason for the condition's last transition.
// 	// +optional
// 	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
// 	// Human readable message indicating details about last transition.
// 	// +optional
// 	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
// }

// type WatcherConditionType string

// const (
// 	WatcherConditionInitializing WatcherConditionType = "Initializing"

// 	WatcherConditionMigratingDatabase WatcherConditionType = "MigratingDatabase"

// 	WatcherConditionStarting WatcherConditionType = "Starting"

// 	WatcherConditionReady WatcherConditionType = "Ready"
// )

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Watcher is the Schema for the watchers API
type Watcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WatcherSpec   `json:"spec,omitempty"`
	Status WatcherStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WatcherList contains a list of Watcher
type WatcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Watcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Watcher{}, &WatcherList{})
}
