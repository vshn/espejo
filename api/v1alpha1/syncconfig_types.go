/*
Licensed under the Apache License, Version 2.0 (the "License");
http://www.apache.org/licenses/LICENSE-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SyncConfigSpec defines the desired state of SyncConfig
type SyncConfigSpec struct {
	// ForceRecreate defines if objects should be deleted and recreated if updates fails
	ForceRecreate bool `json:"forceRecreate,omitempty"`
	// NamespaceSelector defines which namespaces should be targeted
	NamespaceSelector *NamespaceSelector `json:"namespaceSelector,omitempty"`
	// SyncItems lists items to be synced to targeted namespaces
	SyncItems []unstructured.Unstructured `json:"syncItems,omitempty"`
	// DeleteItems lists items to be deleted from targeted namespaces
	DeleteItems []DeleteMeta `json:"deleteItems,omitempty"`
}

// DeleteMeta defines an object by name, kind and version
type DeleteMeta struct {
	// Name of the item to be deleted
	Name string `json:"name,omitempty"`
	// Kind of the item to be deleted
	Kind string `json:"kind,omitempty"`
	// APIVersion of the item to be deleted
	APIVersion string `json:"apiVersion,omitempty"`
}

// NamespaceSelector provides a way to specify targeted namespaces
type NamespaceSelector struct {
	// LabelSelector of namespaces to be targeted
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// MatchNames lists namespace names to be targeted
	MatchNames []string `json:"matchNames,omitempty"`
}

// SyncConfigStatus defines the observed state of SyncConfig
type SyncConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SyncConfig is the Schema for the syncconfigs API
type SyncConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyncConfigSpec   `json:"spec,omitempty"`
	Status SyncConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SyncConfigList contains a list of SyncConfig
type SyncConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyncConfig `json:"items"`
	Prune           bool         `json:"prune,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SyncConfig{}, &SyncConfigList{})
}

// ToDeleteObj creates a k8s Unstructured object based on a DeleteMeta obj
func (d *DeleteMeta) ToDeleteObj(namespace string) *unstructured.Unstructured {
	deleteObj := &unstructured.Unstructured{}
	deleteObj.SetAPIVersion(d.APIVersion)
	deleteObj.SetKind(d.Kind)
	deleteObj.SetName(d.Name)
	deleteObj.SetNamespace(namespace)

	return deleteObj
}
