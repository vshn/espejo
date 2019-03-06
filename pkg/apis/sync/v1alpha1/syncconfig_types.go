package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SyncConfigSpec defines the desired state of SyncConfig
type SyncConfigSpec struct {
	ForceRecreate     bool                        `json:"forceRecreate,omitempty"`
	NamespaceSelector *NamespaceSelector          `json:"namespaceSelector,omitempty"`
	Items             []unstructured.Unstructured `json:"items,omitempty"`
	DeleteItems       []DeleteMeta                `json:"deleteItems,omitempty"`
}

// DeleteMeta defines an object by name, kind and version
type DeleteMeta struct {
	Name       string `json:"name,omitempty"`
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// NamespaceSelector provides a way to specify targeted namespaces
type NamespaceSelector struct {
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	MatchNames    []string              `json:"matchNames,omitempty"`
}

// SyncConfigStatus defines the observed state of SyncConfig
type SyncConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SyncConfig is the Schema for the syncconfigs API
// +k8s:openapi-gen=true
type SyncConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyncConfigSpec   `json:"spec,omitempty"`
	Status SyncConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SyncConfigList contains a list of SyncConfig
type SyncConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Prune           bool         `json:"prune,omitempty"`
	Items           []SyncConfig `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SyncConfig{}, &SyncConfigList{})
}
