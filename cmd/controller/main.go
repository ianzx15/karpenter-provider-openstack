package main

import (
	"github.com/ianzx15/karpenter-provider-openstack/pkg/cloudprovider"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/operator"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/karpenter/pkg/cloudprovider/metrics"
	"sigs.k8s.io/karpenter/pkg/controllers/nodeoverlay"
	"sigs.k8s.io/karpenter/pkg/controllers/state"

	ctrl "sigs.k8s.io/controller-runtime"
	corecontrollers "sigs.k8s.io/karpenter/pkg/controllers"
	coreoperator "sigs.k8s.io/karpenter/pkg/operator"

	v1openstack "github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
)

func main() {

	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	baseCtx, baseOp := coreoperator.NewOperator()

	lo.Must0(v1openstack.AddToScheme(baseOp.Manager.GetScheme()))

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
	instanceTypeStore := nodeoverlay.NewInstanceTypeStore()

	op.WithControllers(ctx, corecontrollers.NewControllers(
		ctx,
		op.Manager,
		op.Clock,
		op.GetClient(),
		op.EventRecorder,
		osCloudProvider,
		cloudProvider,
		clusterState,
		instanceTypeStore,
	)...).Start(ctx)
}
