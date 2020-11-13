package controllers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Replacement(t *testing.T) {
	replacement := "my-string"
	placeholder := "${PROJECT_NAME}"

	var m map[string]interface{}
	m = map[string]interface{}{
		"object-with-nested-objects": map[string]interface{}{
			"object": map[string]interface{}{
				"string-field": placeholder,
			},
		},
		"slice-with-strings":     []string{placeholder},
		"slice-with-other-types": []int{0, 1},
		"slice-with-nested-objects": []map[string]interface{}{
			{
				"object": map[string]interface{}{
					"string-field": placeholder,
				},
			},
			{
				"string-field": placeholder,
			},
		},
		"slice-with-nested-slices": []interface{}{
			[]string{placeholder},
			map[string]interface{}{
				"string-field": placeholder,
				"bool-field":   true,
			},
			[]interface{}{
				[]string{placeholder},
			},
			[]map[string]interface{}{
				{
					"string-field": placeholder,
				},
			},
		},
	}

	replaceProjectName(replacement, m)

	assert.Equal(t, replacement, m["object-with-nested-objects"].(map[string]interface{})["object"].(map[string]interface{})["string-field"])
	assert.Equal(t, replacement, m["slice-with-strings"].([]string)[0])
	assert.Equal(t, replacement, m["slice-with-nested-objects"].([]map[string]interface{})[0]["object"].(map[string]interface{})["string-field"])
	assert.Equal(t, replacement, m["slice-with-nested-objects"].([]map[string]interface{})[1]["string-field"])
	assert.Equal(t, replacement, m["slice-with-nested-slices"].([]interface{})[0].([]string)[0])
	assert.Equal(t, replacement, m["slice-with-nested-slices"].([]interface{})[1].(map[string]interface{})["string-field"])
	assert.Equal(t, replacement, m["slice-with-nested-slices"].([]interface{})[2].([]interface{})[0].([]string)[0])
	assert.Equal(t, replacement, m["slice-with-nested-slices"].([]interface{})[3].([]map[string]interface{})[0]["string-field"])

}
