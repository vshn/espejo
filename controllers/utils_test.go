package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"regexp"
)

var _ = Describe("utils", func() {
	It("should include regular namespace name in selector", func() {
		ns := "match-regular-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}
		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex(ns)}}

		result := includeNamespacesByNames(&rc, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should include namespace that match a pattern", func() {
		ns := "match-regex-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex("match-.*")}}

		result := includeNamespacesByNames(&rc, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should exclude namespace that does not match a pattern", func() {
		ns := "match-regex-namespace"

		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		rc := ReconciliationContext{matchNamesRegex: []*regexp.Regexp{toRegex("match\\snamespaceWithSpace")}}

		result := includeNamespacesByNames(&rc, sourceNs)
		Expect(result).To(BeEmpty())
	})

})

func toRegex(pattern string) *regexp.Regexp {
	rgx, _ := regexp.Compile(pattern)
	return rgx
}
