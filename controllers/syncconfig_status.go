package controllers

import (
	. "github.com/vshn/espejo/api/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *SyncConfigReconciler) updateStatus(rc *ReconciliationContext) error {
	status := rc.cfg.Status
	// Once we are on Kubernetes 0.19, we can use metav1.Conditions, but for now, we have to implement our helpers on
	// our own.
	status.Conditions = mergeWithExistingConditions(status.Conditions, rc.conditions)
	status.SynchronizedItemCount = rc.syncCount
	status.DeletedItemCount = rc.deleteCount
	status.FailedItemCount = rc.failCount

	rc.cfg.Status = status
	err := r.Client.Status().Update(rc.ctx, rc.cfg)
	if err != nil {
		r.Log.Error(err, "Could not update SyncConfig.", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
		return err
	}
	r.Log.WithValues("syncCount", rc.syncCount, "deleteCount", rc.deleteCount, "failCount", rc.failCount).
		Info("Updated SyncConfig status.", getLoggingKeysAndValuesForSyncConfig(rc.cfg)...)
	return nil
}

// SetStatusCondition adds the given condition to the status condition of the SyncConfig. Overwrites existing conditions
// of the same type.
func (rc *ReconciliationContext) SetStatusCondition(condition SyncConfigCondition) {
	rc.conditions[condition.Type] = condition
}

// SetStatusIfExisting sets the condition of the given type to the given status, if the condition already exists, otherwise noop
func (rc *ReconciliationContext) SetStatusIfExisting(conditionType SyncConfigConditionType, status v1.ConditionStatus) {
	if condition, found := rc.conditions[conditionType]; found {
		condition.Status = status
		rc.conditions[conditionType] = condition
	}
}

// CreateStatusConditionReady is a shortcut for adding a SyncConfigReady condition.
func CreateStatusConditionReady(isReady bool) SyncConfigCondition {
	readyCondition := SyncConfigCondition{
		Status:             v1.ConditionFalse,
		Type:               SyncConfigReady,
		LastTransitionTime: metav1.Now(),
		Reason:             SyncReasonFailed,
		Message:            "Synchronization failed",
	}
	if isReady {
		readyCondition.Status = v1.ConditionTrue
		readyCondition.Reason = SyncReasonSucceeded
		readyCondition.Message = "Synchronization completed successfully"
	}
	return readyCondition
}

// CreateStatusConditionErrored is a shortcut for adding a SyncConfigErrored condition with the given error message.
func CreateStatusConditionErrored(err error) SyncConfigCondition {
	return SyncConfigCondition{
		Status:             v1.ConditionTrue,
		Type:               SyncConfigErrored,
		LastTransitionTime: metav1.Now(),
		Reason:             SyncReasonFailedWithError,
		Message:            err.Error(),
	}
}

// CreateStatusConditionInvalid is a shortcut for adding a SyncConfigInvalid condition with the given error message.
func CreateStatusConditionInvalid(err error) SyncConfigCondition {
	return SyncConfigCondition{
		Status:             v1.ConditionTrue,
		Type:               SyncConfigInvalid,
		LastTransitionTime: metav1.Now(),
		Reason:             SyncReasonConfigInvalid,
		Message:            err.Error(),
	}
}

// IncrementSyncCount increments the sync count by 1
func (rc *ReconciliationContext) IncrementSyncCount() {
	rc.syncCount++
}

// IncrementDeleteCount increments the delete count by 1
func (rc *ReconciliationContext) IncrementDeleteCount() {
	rc.deleteCount++
}

// IncrementFailCount increments the fail count by 1
func (rc *ReconciliationContext) IncrementFailCount() {
	rc.failCount++
}

func mapConditionsToType(conditions []SyncConfigCondition) (m SyncConfigConditionMap) {
	m = make(SyncConfigConditionMap)
	for _, c := range conditions {
		m[c.Type] = c
	}
	return m
}

func mergeWithExistingConditions(existing []SyncConfigCondition, new SyncConfigConditionMap) (merged []SyncConfigCondition) {
	exMap := mapConditionsToType(existing)
	for _, condition := range new {
		exMap[condition.Type] = condition
	}
	for _, condition := range exMap {
		merged = append(merged, condition)
	}
	return merged
}
