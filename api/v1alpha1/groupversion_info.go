// Package v1alpha1 contains API Schema definitions for the sync v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=sync.appuio.ch

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "sync.appuio.ch", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
