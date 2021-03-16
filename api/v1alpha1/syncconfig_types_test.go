package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncItem_DeepCopyInto(t *testing.T) {
	configMap := map[string]interface{}{
		"data": map[string]interface{}{
			"KEY": "VALUE",
		},
	}
	item := SyncItem{Object: configMap}
	result := item.DeepCopy()

	assert.Equal(t, item.Object, result.Object)
	assert.Equal(t, &item, result)
	assert.NotSame(t, &item.Object, result.Object)
	assert.NotSame(t, &item, result)
}

func TestSyncItem_DeepCopyInto_ShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		item := SyncItem{}
		item.DeepCopyInto(nil)
	})
}
