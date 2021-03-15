package controllers

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	syncv1alpha1 "github.com/vshn/espejo/api/v1alpha1"
)

func (r *SyncConfigReconciler) updateStatus(rc *ReconciliationContext) error {
	status := rc.cfg.Status
	// Once we are on Kubernetes 0.19, we can use metav1.Conditions, but for now, we have to implement our helpers on
	// our own.
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
func (rc *ReconciliationContext) SetStatusCondition(condition metav1.Condition) {
	meta.SetStatusCondition(&rc.cfg.Status.Conditions, condition)
}

// SetStatusIfExisting sets the condition of the given type to the given status, if the condition already exists, otherwise noop
func (rc *ReconciliationContext) SetStatusIfExisting(conditionType syncv1alpha1.ConditionType, status metav1.ConditionStatus) {
	if condition := meta.FindStatusCondition(rc.cfg.Status.Conditions, conditionType.String()); condition != nil {
		condition.Status = status
		meta.SetStatusCondition(&rc.cfg.Status.Conditions, *condition)
	}
}

// CreateStatusConditionReady is a shortcut for adding a ConditionConfigReady condition.
func CreateStatusConditionReady(isReady bool) metav1.Condition {
	readyCondition := metav1.Condition{
		Status:             metav1.ConditionFalse,
		Type:               syncv1alpha1.ConditionConfigReady.String(),
		LastTransitionTime: metav1.Now(),
		Reason:             syncv1alpha1.SyncReasonFailed,
		Message:            "Synchronization failed",
	}
	if isReady {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Reason = syncv1alpha1.SyncReasonSucceeded
		readyCondition.Message = "Synchronization completed successfully"
	}
	return readyCondition
}

// CreateStatusConditionErrored is a shortcut for adding a ConditionErrored condition with the given error message.
func CreateStatusConditionErrored(err error) metav1.Condition {
	return metav1.Condition{
		Status:  metav1.ConditionTrue,
		Type:    syncv1alpha1.ConditionErrored.String(),
		Reason:  syncv1alpha1.SyncReasonFailedWithError,
		Message: err.Error(),
	}
}

// CreateStatusConditionInvalid is a shortcut for adding a ConditionInvalid condition with the given error message.
func CreateStatusConditionInvalid(err error) metav1.Condition {
	return metav1.Condition{
		Status:             metav1.ConditionTrue,
		Type:               syncv1alpha1.ConditionInvalid.String(),
		LastTransitionTime: metav1.Now(),
		Reason:             syncv1alpha1.SyncReasonConfigInvalid,
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
