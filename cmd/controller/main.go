package main

import (
	"github.com/awslabs/operatorpkg/controller"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/cloudprovider"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/operator"
	"sigs.k8s.io/karpenter/pkg/cloudprovider/metrics"
	"sigs.k8s.io/karpenter/pkg/controllers/state"

	corecontrollers "sigs.k8s.io/karpenter/pkg/controllers"
	coreoperator "sigs.k8s.io/karpenter/pkg/operator"
)

func main() {
	ctx, op := operator.NewOperator(coreoperator.NewOperator())

	osCloudProvider := cloudprovider.New(
		op.GetClient(),
		op.EventRecorder,
		op.InstanceProvider,
		op.InstanceTypeProvider,
	)

	clouProvider := metrics.Decorate(osCloudProvider)

	clusterState := state.NewCluster(op.Clock, op.GetClient(), clouProvider)

	op.WithControllers(ctx, corecontrollers.NewControllers(
		ctx,
		op.Manager,
		op.Clock,
		op.GetClient(),
		op.EventRecorder,
		clouProvider,
		clusterState,

	)...).WithControllers(ctx, controllers

}
