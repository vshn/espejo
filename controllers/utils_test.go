package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("utils", func() {
	It("should include regular namespace name in selector", func() {
		ns := "match-regular-namespace"

		patterns := []string{ns}
		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		result := includeNamespacesByNames(patterns, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should include namespace that match a pattern", func() {
		ns := "match-regex-namespace"

		patterns := []string{"match-.*"}
		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		result := includeNamespacesByNames(patterns, sourceNs)
		Expect(result).To(ConsistOf(sourceNs))
	})

	It("should exclude namespace that does not match a pattern", func() {
		ns := "match-regex-namespace"

		patterns := []string{"match\\snamespaceWithSpace"}
		sourceNs := []v1.Namespace{namespaceFromString(ns)}

		result := includeNamespacesByNames(patterns, sourceNs)
		Expect(result).To(BeEmpty())
	})
})
