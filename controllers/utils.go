package controllers

import (
	"errors"
	"fmt"
	"github.com/vshn/espejo/api/v1alpha1"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

var log = ctrl.Log.WithName("utils")

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
		log.Error(errors.New(fmt.Sprintf("unrecognized type: %s is %T", v, v)), "Cannot replace placeholders in structure")
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
