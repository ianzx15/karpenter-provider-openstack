package instancetype

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
)

type Provider interface {
	List(context.Context, *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error)
}

type DefaultProvider struct {
	InstanceTypesInfo []flavors.Flavor
}

func NewProvider(ctx context.Context, computeClient *gophercloud.ServiceClient) (Provider, error) {
	logger := log.FromContext(ctx)

	flavorPages, err := flavors.ListDetail(computeClient, nil).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %w", err)
	}
	flavorsList, err := flavors.ExtractFlavors(flavorPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract flavors: %w", err)
	}

	logger.Info(fmt.Sprintf("Discovered %d instance types (flavors)", len(flavorsList)))

	return &DefaultProvider{
		InstanceTypesInfo: flavorsList,
	}, nil
}

func (p *DefaultProvider) createOffering() cloudprovider.Offering {
	return cloudprovider.Offering{
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement(
				karpv1.CapacityTypeLabelKey,
				corev1.NodeSelectorOpIn,
				string(karpv1.CapacityTypeOnDemand),
			),
		),
		Available: true,
	}
}

func (p *DefaultProvider) List(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
	instanceTypes := []*cloudprovider.InstanceType{}

	for _, flavor := range p.InstanceTypesInfo {
		maxPods := int64(110)
		if nodeClass.Spec.KubeletConfiguration != nil && nodeClass.Spec.KubeletConfiguration.MaxPods != nil {
			maxPods = int64(*nodeClass.Spec.KubeletConfiguration.MaxPods)
		}

		capacity := corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewQuantity(int64(flavor.VCPUs), resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(int64(flavor.RAM)*1024*1024, resource.BinarySI),
			corev1.ResourcePods:   *resource.NewQuantity(maxPods, resource.DecimalSI),
		}

		offering := p.createOffering()

		instanceType := &cloudprovider.InstanceType{
			Name: flavor.Name,
			Offerings: cloudprovider.Offerings{
				&offering,
			},
			Capacity: capacity,

			Overhead: &cloudprovider.InstanceTypeOverhead{},

			Requirements: scheduling.NewRequirements(
				scheduling.NewRequirement(
					corev1.LabelInstanceTypeStable,
					corev1.NodeSelectorOpIn,
					flavor.Name,
				),
				scheduling.NewRequirement(corev1.LabelArchStable, corev1.NodeSelectorOpIn, "amd64"),
				scheduling.NewRequirement(corev1.LabelOSStable, corev1.NodeSelectorOpIn, "linux"),
			),
		}
		instanceTypes = append(instanceTypes, instanceType)
	}
	return instanceTypes, nil
}
