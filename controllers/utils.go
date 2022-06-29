package controllers

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/vshn/espejo/api/v1alpha1"
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

// replaceProjectName recursively replaces all string occurrences that contain the ${PROJECT_NAME} as placeholder.
// Only replaces the values of objects, does not alter the keys.
func replaceProjectName(replacement string, m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			continue
		}
		m[k] = transcendStructure(replacement, v)
	}
}

func transcendStructure(replacement string, v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case string:
		return replacePlaceholders(replacement, v.(string))
	case []string:
		for i, a := range v.([]string) {
			v.([]string)[i] = replacePlaceholders(replacement, a)
		}
		return v
	case int64:
	case []int64:
	case int32:
	case []int32:
	case int:
	case []int:
	case bool:
	case []bool:
		return v
	case map[string]interface{}:
		for k, m := range v.(map[string]interface{}) {
			v.(map[string]interface{})[k] = transcendStructure(replacement, m)
		}
		return v
	case []map[string]interface{}:
		for k, m := range v.([]map[string]interface{}) {
			v.([]map[string]interface{})[k] = transcendStructure(replacement, m).(map[string]interface{})
		}
		return v
	case []interface{}:
		for i, a := range v.([]interface{}) {
			v.([]interface{})[i] = transcendStructure(replacement, a)
		}
		return v
	default:
		panic(fmt.Errorf("cannot replace placeholders in structure: unrecognized type: %s is %T", v, v))
	}
	return v
}

func replacePlaceholders(replacement, s string) string {
	return strings.ReplaceAll(s, "${PROJECT_NAME}", replacement)
}

func namespaceFromString(namespace string) v1.Namespace {
	return v1.Namespace{
		TypeMeta:   v12.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		ObjectMeta: v12.ObjectMeta{Name: namespace},
	}
}

// copyInto overwrites all non system managed fields of dst with the fields in src.
// This can be used to update dst to the desired version src without creating a diff by removing system managed fields such as UID or SelfLink.
func copyInto(dst, src *unstructured.Unstructured) {
	tmp := dst.DeepCopy()
	src.DeepCopyInto(dst)

	dst.SetResourceVersion(tmp.GetResourceVersion())

	dst.SetUID(tmp.GetUID())
	dst.SetSelfLink(tmp.GetSelfLink())
	dst.SetGeneration(tmp.GetGeneration())

	dst.SetManagedFields(tmp.GetManagedFields())
	dst.SetOwnerReferences(tmp.GetOwnerReferences())

	dst.SetCreationTimestamp(tmp.GetCreationTimestamp())
	dst.SetDeletionTimestamp(tmp.GetDeletionTimestamp())
	dst.SetDeletionGracePeriodSeconds(tmp.GetDeletionGracePeriodSeconds())
}
