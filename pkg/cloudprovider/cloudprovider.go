package cloudprovider

import (
"context"
"fmt"

```
"k8s.io/apimachinery/pkg/api/errors"
"sigs.k8s.io/controller-runtime/pkg/client"

karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
"sigs.k8s.io/karpenter/pkg/cloudprovider"
"sigs.k8s.io/karpenter/pkg/events"

"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1alpha1"
"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instance"
"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instancetype"
```

)

const CloudProviderName = "openstack"

type CloudProvider struct {
kubeClient           client.Client
recorder             events.Recorder
instanceTypeProvider instancetype.Provider
instanceProvider     instance.Provider
}

func New(
kubeClient client.Client,
recorder events.Recorder,
instanceTypeProvider instancetype.Provider,
instanceProvider instance.Provider,
) *CloudProvider {
return &CloudProvider{
kubeClient:           kubeClient,
recorder:             recorder,
instanceTypeProvider: instanceTypeProvider,
instanceProvider:     instanceProvider,
}
}

func (c *CloudProvider) Create(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
nodeClass, err := c.resolveNodeClassFromNodeClaim(ctx, nodeClaim)
if err != nil {
if errors.IsNotFound(err) {
c.recorder.Publish(events.Event{
Type:    "Warning",
Reason:  "NodeClassNotFound",
Message: fmt.Sprintf("NodeClass %s not found", nodeClaim.Spec.NodeClassRef.Name),
})
}
return nil, fmt.Errorf("resolving node class: %w", err)
}

instanceTypes, err := c.instanceTypeProvider.List(ctx, nodeClass)
if err != nil || len(instanceTypes) == 0 {
	return nil, fmt.Errorf("no instance types available: %w", err)
}

inst, err := c.instanceProvider.Create(ctx, nodeClass, nodeClaim, instanceTypes)
if err != nil {
	return nil, fmt.Errorf("creating OpenStack instance: %w", err)
}

newNodeClaim := &karpv1.NodeClaim{}
newNodeClaim.Status.ProviderID = fmt.Sprintf("openstack://%s/%s", inst.ProjectID, inst.Name)
return newNodeClaim, nil

}

func (c *CloudProvider) resolveNodeClassFromNodeClaim(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*v1alpha1.OpenStackNodeClass, error) {
ref := nodeClaim.Spec.NodeClassRef
if ref == nil {
return nil, fmt.Errorf("nodeClaim missing NodeClassRef")
}

nodeClass := &v1alpha1.OpenStackNodeClass{}
if err := c.kubeClient.Get(ctx, client.ObjectKey{Name: ref.Name}, nodeClass); err != nil {
	return nil, err
}

return nodeClass, nil

}
