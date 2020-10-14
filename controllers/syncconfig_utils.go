package controllers

import (
	"fmt"
	"github.com/vshn/espejo/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"regexp"
)

func (r *SyncConfigReconciler) validateSpec(rc *ReconciliationContext) error {
	spec := rc.cfg.Spec
	if hasNoNamespaceSelector(rc.cfg.Spec) {
		return fmt.Errorf("either .spec.namespaceSelector.matchNames or .spec.namespaceSelector.labelSelector is required")
	}
	if len(spec.DeleteItems) == 0 && len(spec.SyncItems) == 0 {
		return fmt.Errorf("either spec.deleteItems or .spec.syncItems is required")
	}
	for _, pattern := range spec.NamespaceSelector.MatchNames {
		rgx, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf(".spec.namespaceSelector.matchNames pattern invalid: %w", err)
		}
		rc.matchNamesRegex = append(rc.matchNamesRegex, rgx)
	}
	for _, pattern := range spec.NamespaceSelector.IgnoreNames {
		rgx, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf(".spec.namespaceSelector.ignoreNames pattern invalid: %w", err)
		}
		rc.ignoreNamesRegex = append(rc.ignoreNamesRegex, rgx)
	}
	if rc.cfg.Spec.NamespaceSelector.LabelSelector != nil {
		labelSelector, err := metav1.LabelSelectorAsSelector(rc.cfg.Spec.NamespaceSelector.LabelSelector)
		if err != nil {
			return fmt.Errorf(".spec.namespaceSelector.labelSelector is invalid: %w", err)
		}
		rc.nsSelector = labelSelector
	}

	return nil
}

func filterNamespaces(rc *ReconciliationContext, namespaceList []v1.Namespace) (namespaces []v1.Namespace) {
NamespaceLoop:
	for _, ns := range namespaceList {
		for _, regex := range rc.ignoreNamesRegex {
			if regex.MatchString(ns.Name) {
				continue NamespaceLoop
			}
		}
		if rc.nsSelector != nil && rc.nsSelector.Matches(labels.Set(ns.GetLabels())) {
			namespaces = append(namespaces, ns)
			continue NamespaceLoop
		}
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

// hasNoNamespaceSelector will return true if the SyncConfigSpec does not have a valid namespace selector
func hasNoNamespaceSelector(spec v1alpha1.SyncConfigSpec) bool {
	if spec.NamespaceSelector == nil {
		return true
	}
	return spec.NamespaceSelector.LabelSelector == nil && len(spec.NamespaceSelector.MatchNames) == 0
}
