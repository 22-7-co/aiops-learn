/*
Copyright 2026.

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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronHPASpec defines the desired state of CronHPA
type CronHPASpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of CronHPA. Edit cronhpa_types.go to remove/update
	// +optional
	Foo *string `json:"foo,omitempty"`

	ScaleTargetRef ScaleTargetRefrence `json:"scaleTargetRef"`
	// 扩缩容任务
	Jobs []JobSpec `json:"jobs"`
}

type ScaleTargetRefrence struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type JobSpec struct {
	Name       string `json:"name"`
	Schedule   string `json:"schedule"`
	TargetSize int32  `json:"targetSize"`
}

// CronHPAStatus defines the observed state of CronHPA.
type CronHPAStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the CronHPA resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// 副本数
	CurrentReplicas int32 `json:"currentReplicas"`
	// 最后一次扩容时间
	LastScaleTime *metav1.Time `json:"lastScaleTime"`
	// 最后一次运行时间
	LastRunTime map[string]metav1.Time `json:"lastRunTime"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="Target",type="string",JSONPath=".spec.scaleTargetRef.name",description="目标工作负载"
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.jobs[*].schedule",description="Cron调度规则"
// +kubebuilder:printcolumn:name="Target Size",type="string",JSONPath=".status.conditions[*].targetSize",description="目标副本数"
// CronHPA is the Schema for the cronhpas API
type CronHPA struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of CronHPA
	// +required
	Spec CronHPASpec `json:"spec"`

	// status defines the observed state of CronHPA
	// +optional
	Status CronHPAStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CronHPAList contains a list of CronHPA
type CronHPAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []CronHPA `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronHPA{}, &CronHPAList{})
}
