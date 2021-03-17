package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type (
	// SyncConfigSpec defines the desired state of SyncConfig
	SyncConfigSpec struct {
		// ForceRecreate defines if objects should be deleted and recreated if updates fails
		ForceRecreate bool `json:"forceRecreate,omitempty"`
		// NamespaceSelector defines which namespaces should be targeted
		NamespaceSelector *NamespaceSelector `json:"namespaceSelector,omitempty"`
		// SyncItems lists items to be synced to targeted namespaces
		SyncItems []SyncItem `json:"syncItems,omitempty"`
		// DeleteItems lists items to be deleted from targeted namespaces
		DeleteItems []DeleteMeta `json:"deleteItems,omitempty"`
	}

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:validation:XEmbeddedResource

	// SyncItem is an unstructured, "free-form" Kubernetes resource, complete with GVK, metadata and spec.
	SyncItem unstructured.Unstructured

	// DeleteMeta defines an object by name, kind and version
	DeleteMeta struct {
		// Name of the item to be deleted
		Name string `json:"name,omitempty"`
		// Kind of the item to be deleted
		Kind string `json:"kind,omitempty"`
		// APIVersion of the item to be deleted
		APIVersion string `json:"apiVersion,omitempty"`
	}

	// NamespaceSelector provides a way to specify targeted namespaces
	NamespaceSelector struct {
		// LabelSelector of namespaces to be targeted. Can be combined with MatchNames to include unlabelled namespaces.
		LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
		// MatchNames lists namespace names to be targeted. Each entry can be a Regex pattern.
		// A namespace is included if at least one pattern matches.
		// Invalid patterns will cause the sync to be cancelled and the status conditions will contain the error message.
		MatchNames []string `json:"matchNames,omitempty"`
		// IgnoreNames lists namespace names to be ignored. Each entry can be a Regex pattern and if they match
		// the namespaces will be excluded from the sync even if matching in "matchNames" or via LabelSelector.
		// A namespace is ignored if at least one pattern matches.
		// Invalid patterns will cause the sync to be cancelled and the status conditions will contain the error message.
		IgnoreNames []string `json:"ignoreNames,omitempty"`
	}

	// SyncConfigStatus defines the observed state of SyncConfig
	SyncConfigStatus struct {
		// Conditions contain the states of the SyncConfig. A SyncConfig is considered Ready when at least one item has been synced.
		Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge"`
		// SynchronizedItemCount holds the accumulated number of created or updated objects in the targeted namespaces.
		SynchronizedItemCount int64 `json:"synchronizedItemCount"`
		// DeletedItemCount holds the accumulated number of deleted objects from targeted namespaces. Inexisting items do not get counted.
		DeletedItemCount int64 `json:"deletedItemCount"`
		// FailedItemCount holds the accumulated number of objects that could not be created, updated or deleted. Inexisting items do not get counted.
		FailedItemCount int64 `json:"failedItemCount"`
	}

	// ConditionType identifies the type of a condition. The type is unique in the Status field.
	ConditionType string

	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	// +kubebuilder:printcolumn:name="Synced",type=integer,JSONPath=`.status.synchronizedItemCount`
	// +kubebuilder:printcolumn:name="Deleted",type=integer,JSONPath=`.status.deletedItemCount`
	// +kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.failedItemCount`
	// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

	// SyncConfig is the Schema for the syncconfigs API
	SyncConfig struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   SyncConfigSpec   `json:"spec,omitempty"`
		Status SyncConfigStatus `json:"status,omitempty"`
	}

	// +kubebuilder:object:root=true

	// SyncConfigList contains a list of SyncConfig
	SyncConfigList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []SyncConfig `json:"items"`
		Prune           bool         `json:"prune,omitempty"`
	}
)

const (
	// ConditionConfigReady tracks if the SyncConfig has been successfully reconciled.
	ConditionConfigReady ConditionType = "Ready"
	// ConditionErrored is given when no objects could be synced or deleted and the failed object count is > 0 or
	// any other reconciliation error.
	ConditionErrored ConditionType = "Errored"
	// ConditionInvalid is given when the the SyncConfig Spec contains invalid properties. SyncConfigs will not be
	// reconciled.
	ConditionInvalid ConditionType = "Invalid"

	// SyncReasonFailed is given when the sync generally failed.
	SyncReasonFailed = "SynchronizationFailed"
	// SyncReasonSucceeded is given when the sync succeeded without errors.
	SyncReasonSucceeded = "SynchronizationSucceeded"
	// SyncReasonFailedWithError is given when the sync failed with a particular error.
	SyncReasonFailedWithError = "SynchronizationFailedWithError"
	// SyncReasonConfigInvalid is given if the SyncConfig contains invalid spec.
	SyncReasonConfigInvalid = "InvalidSyncConfigSpec"
)

func init() {
	SchemeBuilder.Register(&SyncConfig{}, &SyncConfigList{})
}

// ToDeleteObj creates a k8s Unstructured object based on a DeleteMeta obj
func (in *DeleteMeta) ToDeleteObj(namespace string) *unstructured.Unstructured {
	deleteObj := &unstructured.Unstructured{}
	deleteObj.SetAPIVersion(in.APIVersion)
	deleteObj.SetKind(in.Kind)
	deleteObj.SetName(in.Name)
	deleteObj.SetNamespace(namespace)

	return deleteObj
}

// String returns string(condition).
func (in ConditionType) String() string {
	return string(in)
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncItem) DeepCopyInto(out *SyncItem) {
	// controller-gen cannot handle the interface{} type of an aliased Unstructured, thus we write our own DeepCopyInto function.
	if out != nil {
		casted := unstructured.Unstructured(*in)
		deepCopy := casted.DeepCopy()
		out.Object = deepCopy.Object
	}
}
