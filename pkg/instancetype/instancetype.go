package instancetype

import (
	"context"
	"net/http"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	corev1 "k8s.io/api/core/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
)

type Provider interface {
	LivenessProbe(*http.Request) error
	List(context.Context, *v1openstack.OSNodeClassSpec) ([]*cloudprovider.InstanceType, error)
}

type DefaultProvider struct {
}

func (p *DefaultProvider) createOffering(capacityType string, available bool) *cloudprovider.Offering {
	return &cloudprovider.Offering{
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement(karpv1.CapacityTypeLabelKey, corev1.NodeSelectorOpIn, capacityType),
		),
		Available: available,
	}
}

func (p *DefaultProvider) List(ctx context.Context, nodeClass *v1openstack.OSNodeClass) ([]*cloudprovider.InstanceType, error) {
	reqs := scheduling.NewRequirements(
		scheduling.NewRequirement(corev1.LabelInstanceTypeStable, selection.LabelSelectorOpIn, "m1.small"),
	)

	instanceType := &cloudprovider.InstanceType{}

	return []*cloudprovider.InstanceType{instanceType}, nil
}
