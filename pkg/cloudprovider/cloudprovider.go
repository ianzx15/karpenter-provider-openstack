package cloudprovider

import{

	"context"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	"sigs.k8s.io/karpenter/pkg/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"


}


type CloudProvider struct{
	kubeClient client.Client
	recorder events.Recorder
	
	instanceProvider instance.Provider
}

func New(kubeClient client.Client, recorder events.Recorder,
	instanceProvider instance.Provider) *CloudProvider {
		return &CloudProvider{
			kubeClient:      kubeClient,
			recorder:        recorder,
			instanceProvider: instanceProvider,
		}
	}



func (c *CloudProvider) Create(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
	nodeClass, err := c.resolveNodeClassfromNodeClaim(ctx, nodeClaim)
	if err != nil {
		return nil, err
	}

	instance, err := c.instanceProvider.Create(ctx, nodeClass, nodeClaim)


}


/*
TODO
resolveInstanceTypeFromInstance ? Talvez nao seja necessario
resolveNodePoolFromInstance
resolveNodeClassFromNodePool
resolveNodeClassFromNodeClaim ?
instanceToNodeClaim
*/