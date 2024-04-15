/*
Copyright 2023.

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
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ScavengerJobSpec defines the desired state of ScavengerJob
type ScavengerJobSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	Job batch.JobSpec `json:"job"`
}

// ScavengerJobStatus defines the observed state of ScavengerJob
type ScavengerJobStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	Status                 ScavengerJobStatusType `json:"status,omitempty"`
	StartTime              *metav1.Time           `json:"startTime,omitempty"`
	InterruptionTimeStamps []metav1.Time          `json:"interruptionTimeStamps,omitempty"`
	RunningTimeStamps      []metav1.Time          `json:"runningTimeStamps,omitempty"`
	CompletionTimeStamp    *metav1.Time           `json:"completionTimeStamp,omitempty"`
	LastCheckpoint         *metav1.Time           `json:"lastCheckpoint,omitempty"`
	Message                string                 `json:"message,omitempty"`
}

type ScavengerJobStatusType string

const (
	ScavengerJobStatusTypeNotStarted  ScavengerJobStatusType = "Not Started"
	ScavengerJobStatusTypePending     ScavengerJobStatusType = "Pending"
	ScavengerJobStatusTypeRunning     ScavengerJobStatusType = "Running"
	ScavengerJobStatusTypeInterrupted ScavengerJobStatusType = "Interrupted"
	ScavengerJobStatusTypeCompleted   ScavengerJobStatusType = "Completed"
	ScavengerJobStatusTypeFailing     ScavengerJobStatusType = "Failing"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ScavengerJob is the Schema for the scavengerjobs API
type ScavengerJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScavengerJobSpec   `json:"spec"`
	Status ScavengerJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ScavengerJobList contains a list of ScavengerJob
type ScavengerJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScavengerJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScavengerJob{}, &ScavengerJobList{})
}
