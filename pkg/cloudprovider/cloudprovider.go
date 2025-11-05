package cloudprovider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/awslabs/operatorpkg/status"
	"github.com/cloudpilot-ai/karpenter-provider-gcp/pkg/utils"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instancetype"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/events"
	"sigs.k8s.io/karpenter/pkg/scheduling"
	"sigs.k8s.io/karpenter/pkg/utils/resources"
)

var _ cloudprovider.CloudProvider = (*CloudProvider)(nil)

type CloudProvider struct {
	kubeClient client.Client
	recorder   events.Recorder

	instanceTypeProvider instancetype.Provider
	instanceProvider     instance.Provider
}

func New(kubeClient client.Client, recorder events.Recorder,
	instanceProvider instance.Provider, instanceTypeProvider instancetype.Provider) *CloudProvider {
	return &CloudProvider{
		kubeClient:       kubeClient,
		recorder:         recorder,
		instanceProvider: instanceProvider,
		instanceTypeProvider: instanceTypeProvider,

	}
}

func (c *CloudProvider) Create(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
	nodeClass, err := c.resolveNodeClassFromNodeClaim(ctx, nodeClaim)
	if err != nil {
		return nil, err
	}

	instancetypes, err := c.resolveInstanceTypes(ctx, nodeClaim, nodeClass)
	if err != nil {
		return nil, err
	}

	if len(instancetypes) == 0 {
		return nil, fmt.Errorf("no instance types available for the given NodeClaim requirements")
	}

	instance, err := c.instanceProvider.Create(ctx, nodeClass, nodeClaim, instancetypes)

	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}

	instanceType, _ := lo.Find(instancetypes, func(it *cloudprovider.InstanceType) bool {
		return it.Name == instance.Type
	})

	nc := c.instanceToNodeClaim(instance, instanceType)

	return nc, nil
}
func (c *CloudProvider) resolveInstanceTypes(ctx context.Context, nodeClaim *karpv1.NodeClaim, nodeClass *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
	instanceTypes, err := c.instanceTypeProvider.List(ctx, nodeClass)
	if err != nil {
		return nil, fmt.Errorf("listing instance types: %w", err)
	}

	reqs := scheduling.NewNodeSelectorRequirementsWithMinValues(nodeClaim.Spec.Requirements...)
	return lo.Filter(instanceTypes, func(i *cloudprovider.InstanceType, _ int) bool {
		return reqs.Compatible(i.Requirements, scheduling.AllowUndefinedWellKnownLabels) == nil &&
			len(i.Offerings.Compatible(reqs).Available()) > 0 &&
			resources.Fits(nodeClaim.Spec.Resources.Requests, i.Allocatable())
	}), nil
}

func (c *CloudProvider) resolveNodeClassFromNodeClaim(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*v1openstack.OpenStackNodeClass, error) {
	ref := nodeClaim.Spec.NodeClassRef
	if ref == nil {
		return nil, fmt.Errorf("nodeClaim missing NodeClassRef")
	}
	nodeClass := &v1openstack.OpenStackNodeClass{}
	if err := c.kubeClient.Get(ctx, types.NamespacedName{Name: ref.Name}, nodeClass); err != nil {
		return nil, fmt.Errorf("getting NodeClass %s/%s: %w", ref.Name, ref.Name, err)
	}
	return nodeClass, nil
}

func (c *CloudProvider) instanceToNodeClaim(instance *instance.Instance, instanceType *cloudprovider.InstanceType) *karpv1.NodeClaim {
	nodeClaim := &karpv1.NodeClaim{}
	labels := map[string]string{}
	annotations := map[string]string{}

	if instanceType != nil {
		labels = utils.GetAllSingleValuedRequirementLabels(instanceType)

		resourceFilter := func(name corev1.ResourceName, value resource.Quantity) bool {
			return !resources.IsZero(value)
		}
		nodeClaim.Status.Capacity = lo.PickBy(instanceType.Capacity, resourceFilter)
		nodeClaim.Status.Allocatable = lo.PickBy(instanceType.Allocatable(), resourceFilter)
	}

	labels["instance-type"] = instance.Type

	nodeClaim.ObjectMeta.Labels = labels
	nodeClaim.ObjectMeta.Annotations = annotations

	nodeClaim.Status.ProviderID = fmt.Sprintf("openstack://%s/%s", instance.InstanceID, instance.Name)
	nodeClaim.Status.ImageID = instance.ImageID

	return nodeClaim
}

func (c *CloudProvider) Delete(ctx context.Context, nodeClaim *karpv1.NodeClaim) error {
	return nil
}

func (c *CloudProvider) List(ctx context.Context) ([]*karpv1.NodeClaim, error) {
	return nil, nil
}

func (c *CloudProvider) Get(ctx context.Context, providerID string) (*karpv1.NodeClaim, error) {
	return nil, nil
}

func (c *CloudProvider) GetInstanceTypes(ctx context.Context, nodePool *karpv1.NodePool) ([]*cloudprovider.InstanceType, error) {
	return nil, nil
}

func (c *CloudProvider) IsDrifted(ctx context.Context, nodeClaim *karpv1.NodeClaim) (cloudprovider.DriftReason, error) {
	return "", nil
}

func (c *CloudProvider) RepairPolicies() []cloudprovider.RepairPolicy {
	return nil
}

func (c *CloudProvider) Name() string {
	return "openstack"
}

func (c *CloudProvider) GetSupportedNodeClasses() []status.Object {
	return nil
}

func (c *CloudProvider) LivenessProbe(req *http.Request) error {
	return nil
}
