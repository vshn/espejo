package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vshn/espejo/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"regexp"
)

var _ = Describe("SyncConfig utils", func() {
	It("should include regular namespace name in selector", func() {
		ns := "match-regular-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}
		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex(ns)}}

		result := filterNamespaces(&rc, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should include namespace that match a pattern", func() {
		ns := "match-regex-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex("match-.*")}}

		result := filterNamespaces(&rc, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should exclude namespace that does not match a pattern", func() {
		ns := "match-regex-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex("match\\snamespaceWithSpace")}}

		result := filterNamespaces(&rc, sourceNs)
		Expect(result).To(BeEmpty())
	})

	It("should fail validation if invalid regex is specified in matchNames", func() {
		cfg := &v1alpha1.SyncConfig{Spec: v1alpha1.SyncConfigSpec{
			NamespaceSelector: &v1alpha1.NamespaceSelector{
				MatchNames: []string{"["},
			},
			SyncItems: []unstructured.Unstructured{toUnstructured(&v1.ConfigMap{})},
		}}
		rc := ReconciliationContext{cfg: cfg}

		err := rc.validateSpec()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error parsing regexp"))
	})

	It("should fail validation if invalid regex is specified in ignoreNames", func() {
		cfg := &v1alpha1.SyncConfig{Spec: v1alpha1.SyncConfigSpec{
			NamespaceSelector: &v1alpha1.NamespaceSelector{
				IgnoreNames: []string{"["},
				MatchNames:  []string{".*"},
			},
			SyncItems: []unstructured.Unstructured{toUnstructured(&v1.ConfigMap{})},
		}}
		rc := ReconciliationContext{cfg: cfg}

		err := rc.validateSpec()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error parsing regexp"))
	})

	It("should fail validation if no namespace selector is given", func() {
		cfg := &v1alpha1.SyncConfig{Spec: v1alpha1.SyncConfigSpec{
			NamespaceSelector: &v1alpha1.NamespaceSelector{
				MatchNames: []string{},
			},
		}}
		rc := ReconciliationContext{cfg: cfg}

		err := rc.validateSpec()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("labelSelector is required"))
	})

	It("should not include namespace name that shares the same substring with another", func() {
		ns1 := "default"
		ns2 := "substring-with-default"

		cfg := &v1alpha1.SyncConfig{Spec: v1alpha1.SyncConfigSpec{
			NamespaceSelector: &v1alpha1.NamespaceSelector{
				MatchNames: []string{ns1},
			},
			SyncItems: []unstructured.Unstructured{toUnstructured(&v1.ConfigMap{})},
		}}

		sourceNs := []v1.Namespace{namespaceFromString(ns1), namespaceFromString(ns2)}
		rc := ReconciliationContext{cfg: cfg}
		err := rc.validateSpec()
		Expect(err).ToNot(HaveOccurred())

		result := filterNamespaces(&rc, sourceNs)
		Expect(result).To(ContainElement(namespaceFromString(ns1)))
		Expect(result).ToNot(ContainElement(namespaceFromString(ns2)))
	})

})

func toRegex(pattern string) *regexp.Regexp {
	rgx, _ := regexp.Compile(pattern)
	return rgx
}
