package controllers

import (
	"github.com/vshn/espejo/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getLoggingKeysAndValues(unstructuredObject *unstructured.Unstructured) []interface{} {
	return []interface{}{
		"Object.Kind", unstructuredObject.GetKind(),
		"Object.Namespace", unstructuredObject.GetNamespace(),
		"Object.Name", unstructuredObject.GetName(),
	}
}

func getLoggingKeysAndValuesForSyncConfig(syncconfig *v1alpha1.SyncConfig) []interface{} {
	return []interface{}{
		"SyncConfig", syncconfig.Namespace + "/" + syncconfig.Name,
	}
}
