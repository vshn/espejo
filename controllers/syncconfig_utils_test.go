package controllers

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
)

func Test_ReconciliationContext_FilterNamespaces(t *testing.T) {
	tests := map[string]struct {
		givenNamespaces      []corev1.Namespace
		givenMatchNamesRegex []*regexp.Regexp
		expectedNamespaces   []corev1.Namespace
	}{
		"GivenRegularNameAsRegex_WhenFilter_ThenIncludeFullName": {
			givenNamespaces:      []corev1.Namespace{namespaceFromString("match-regular-namespace")},
			givenMatchNamesRegex: []*regexp.Regexp{toRegex(t, "match-regular-namespace")},
			expectedNamespaces:   []corev1.Namespace{namespaceFromString("match-regular-namespace")},
		},
		"GivenNamePattern_WhenFilter_ThenIncludeFullName": {
			givenNamespaces:      []corev1.Namespace{namespaceFromString("match-regex-namespace")},
			givenMatchNamesRegex: []*regexp.Regexp{toRegex(t, "match-.*")},
			expectedNamespaces:   []corev1.Namespace{namespaceFromString("match-regex-namespace")},
		},
		"GivenNonMatchingPattern_WhenFilter_ThenIgnore": {
			givenNamespaces:      []corev1.Namespace{namespaceFromString("match-regex-namespace")},
			givenMatchNamesRegex: []*regexp.Regexp{toRegex(t, "match\\snamespaceWithSpace")},
			expectedNamespaces:   []corev1.Namespace{},
		},
		"GivenRegularNamespace_WhenFiltering_ThenReturnSameNamespace": {
			givenMatchNamesRegex: []*regexp.Regexp{toRegex(t, "match-regular-namespace")},
			givenNamespaces:      []corev1.Namespace{namespaceFromString("match-regular-namespace")},
			expectedNamespaces:   []corev1.Namespace{namespaceFromString("match-regular-namespace")},
		},
		"GivenNamespaceThatSharesSameSubString_WhenFiltering_ThenIgnoreSimilarNamespace": {
			givenMatchNamesRegex: []*regexp.Regexp{toRegex(t, "default")},
			givenNamespaces: []corev1.Namespace{
				namespaceFromString("default"),
				namespaceFromString("substring-with-default"),
			},
			expectedNamespaces: []corev1.Namespace{namespaceFromString("default")},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc := ReconciliationContext{matchNamesRegex: tt.givenMatchNamesRegex}
			filtered := rc.filterNamespaces(tt.expectedNamespaces)
			assert.Equal(t, tt.expectedNamespaces, filtered)
		})
	}
}

func Test_ReconciliationContext_validateSpec(t *testing.T) {
	tests := map[string]struct {
		expectErr          bool
		containsErrMessage string

		cfg              *syncv1alpha1.SyncConfig
		matchNamesRegex  []*regexp.Regexp
		ignoreNamesRegex []*regexp.Regexp
		nsSelector       labels.Selector
	}{
		"GivenSpecWithInvalidMatchNamesSelector_WhenParsingRegex_ThenReturnRegexError": {
			cfg: &syncv1alpha1.SyncConfig{
				Spec: syncv1alpha1.SyncConfigSpec{
					NamespaceSelector: &syncv1alpha1.NamespaceSelector{
						MatchNames: []string{"["},
					},
					SyncItems: []syncv1alpha1.Manifest{{Unstructured: toUnstructured(t, &corev1.ConfigMap{})}},
				},
			},
			containsErrMessage: "error parsing regexp",
			expectErr:          true,
		},
		"GivenSpecWithInvalidIgnoreNamesSelector_WhenParsingRegex_ThenReturnRegexError": {
			cfg: &syncv1alpha1.SyncConfig{
				Spec: syncv1alpha1.SyncConfigSpec{
					NamespaceSelector: &syncv1alpha1.NamespaceSelector{
						IgnoreNames: []string{"["},
						MatchNames:  []string{".*"},
					},
					SyncItems: []syncv1alpha1.Manifest{{Unstructured: toUnstructured(t, &corev1.ConfigMap{})}},
				},
			},
			containsErrMessage: "error parsing regexp",
			expectErr:          true,
		},
		"GivenSpecWithNoNamespaceSelector_WhenValidating_ThenReturnSelectorError": {
			cfg: &syncv1alpha1.SyncConfig{
				Spec: syncv1alpha1.SyncConfigSpec{
					NamespaceSelector: &syncv1alpha1.NamespaceSelector{
						MatchNames: []string{},
					},
				},
			},
			containsErrMessage: "labelSelector is required",
			expectErr:          true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc := &ReconciliationContext{
				cfg: tt.cfg,
			}
			err := rc.validateSpec()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.containsErrMessage)
				return
			}

		})
	}
}

func toRegex(t *testing.T, pattern string) *regexp.Regexp {
	rgx, err := regexp.Compile(pattern)
	require.NoError(t, err)
	return rgx
}

func toUnstructured(t *testing.T, obj *corev1.ConfigMap) unstructured.Unstructured {
	converted := unstructured.Unstructured{}
	o := obj.DeepCopy()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	require.NoError(t, err)
	converted.SetUnstructuredContent(m)
	return converted
}

func toObjectMeta(name, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}
