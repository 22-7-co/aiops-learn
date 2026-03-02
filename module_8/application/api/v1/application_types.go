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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of Application. Edit application_types.go to remove/update
	// +optional
	// Foo *string `json:"foo,omitempty"`
	Deployment ApplicationDeployment    `json:"deployment"`
	Service    corev1.ServiceSpec       `json:"service"`
	Ingress    networkingv1.IngressSpec `json:"ingress"`
}

type ApplicationDeployment struct {
	Image    string             `json:"image"`
	Replicas int32              `json:"replicas"`
	Port     int32              `json:"port"`
	Service  corev1.ServiceSpec `json:"service"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	AvailableReplicas int32 `json:"availableReplicas"`
	// Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applications,scope=Namespaced
// +kubebuilder:subresource:status

// Application is the Schema for the applications API
// type Application struct {
// 	metav1.TypeMeta `json:",inline"`

// 	// metadata is a standard object metadata
// 	// +optional
// 	metav1.ObjectMeta `json:"metadata,omitzero"`

// 	// spec defines the desired state of Application
// 	// +required
// 	Spec ApplicationSpec `json:"spec"`

//		// status defines the observed state of Application
//		// +optional
//		Status ApplicationStatus `json:"status,omitzero"`
//	}
// type Application struct {
// 	metav1.TypeMeta `json:",inline"`

// 	// metadata is a standard object metadata
// 	// +optional
// 	metav1.ObjectMeta `json:"metadata,omitempty"` // ← 改成 omitempty

// 	// spec defines the desired state of Application
// 	// +required
// 	Spec ApplicationSpec `json:"spec"`

//		// status defines the observed state of Application
//		// +optional
//		Status ApplicationStatus `json:"status,omitempty"` // ← 改成 omitempty
//	}
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// type Application struct {
// 	metav1.TypeMeta   `json:",inline"`
// 	metav1.ObjectMeta `json:"metadata,omitempty"` // 注意：必须是内嵌，不能是字段名
// 	Spec              ApplicationSpec             `json:"spec,omitempty"`
// 	Status            ApplicationStatus           `json:"status,omitempty"`
// }

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
