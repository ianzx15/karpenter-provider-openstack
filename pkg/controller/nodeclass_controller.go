package controller

import (
	"context"
	"fmt"

	"github.com/awslabs/operatorpkg/status"
	v1openstack "github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OpenStackNodeClassReconciler struct {
	Client client.Client
}

func (r *OpenStackNodeClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nodeClass := &v1openstack.OpenStackNodeClass{}
	if err := r.Client.Get(ctx, req.NamespacedName, nodeClass); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	conditionSet := nodeClass.StatusConditions()

	conditionSet.SetTrueWithReason(
		status.ConditionReady,
		"BypassedValidation",
		"Status forced to Ready: True for debugging.",
	)
	if err := r.Client.Status().Update(ctx, nodeClass); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating OpenStackNodeClass status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackNodeClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1openstack.OpenStackNodeClass{}).
		Complete(r)
}
