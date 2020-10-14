package controllers

import (
	"github.com/vshn/espejo/api/v1alpha1"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
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

func replaceProjectName(replacement string, m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			continue
		}
		switch v.(type) {
		case string:
			s := m[k].(string)
			m[k] = strings.ReplaceAll(s, "${PROJECT_NAME}", replacement)
		case int64:
		case int32:
		case int:
		case bool:
			continue
		case []interface{}:
			for _, elem := range v.([]interface{}) {
				replaceProjectName(replacement, elem.(map[string]interface{}))
			}
		case interface{}:
			replaceProjectName(replacement, m[k].(map[string]interface{}))
		}
	}
}

func namespaceFromString(namespace string) v1.Namespace {
	return v1.Namespace{
		TypeMeta:   v12.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		ObjectMeta: v12.ObjectMeta{Name: namespace},
	}
}

func includeNamespacesByNames(rc *ReconciliationContext, namespaceList []v1.Namespace) (namespaces []v1.Namespace) {
NamespaceLoop:
	for _, ns := range namespaceList {
		for _, regex := range rc.matchNamesRegex {
			if regex.MatchString(ns.Name) {
				namespaces = append(namespaces, ns)
				continue NamespaceLoop
			}
		}
	}
	return namespaces
}

// isReconcileFailed returns true if no objects could be synced or deleted and failedCount is > 0
func isReconcileFailed(rc *ReconciliationContext) bool {
	return rc.syncCount == 0 && rc.deleteCount == 0 && rc.failCount > 0
}
