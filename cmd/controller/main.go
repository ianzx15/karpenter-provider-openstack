package main

import (
	"github.com/ianzx15/karpenter-provider-openstack/pkg/cloudprovider"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/operator"
	"github.com/samber/lo"
	"sigs.k8s.io/karpenter/pkg/cloudprovider/metrics"
	"sigs.k8s.io/karpenter/pkg/controllers/state"

	corecontrollers "sigs.k8s.io/karpenter/pkg/controllers"
	coreoperator "sigs.k8s.io/karpenter/pkg/operator"
)

func main() {
	baseCtx, baseOp := coreoperator.NewOperator()

	ctx, op := operator.NewOperator(baseCtx, baseOp)

	osCloudProvider := cloudprovider.New(
		op.GetClient(),
		op.EventRecorder,
		op.InstanceProvider,
		op.InstanceTypeProvider,
	)
	lo.Must0(op.AddHealthzCheck("cloud-provider", osCloudProvider.LivenessProbe))

	cloudProvider := metrics.Decorate(osCloudProvider)

	clusterState := state.NewCluster(op.Clock, op.GetClient(), cloudProvider)

	op.WithControllers(ctx, corecontrollers.NewControllers(
		ctx,
		op.Manager,
		op.Clock,
		op.GetClient(),
		op.EventRecorder,
		cloudProvider,
		clusterState,
	)...).Start(ctx)
}
