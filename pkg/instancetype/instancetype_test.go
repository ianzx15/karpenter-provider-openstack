package instancetype

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
)

func TestListInstanceTypes(t *testing.T) {
	mockFlavors := []*flavors.Flavor{
		{
			Name:  "general.small",
			VCPUs: 2,
			RAM:   4096,
		},
		{
			Name:  "medium",
			VCPUs: 4,
			RAM:   8192,
		},
	}

	provider := DefaultProvider{
		InstanceTypesInfo: mockFlavors,
	}

	nodeClass := &v1openstack.OpenStackNodeClass{
		Spec: v1openstack.OpenStackNodeClassSpec{
			Flavor: "genarl.small",
			KubeletConfiguration: &v1openstack.KubeletConfiguration{
				MaxPods: nil,
			},
		},
	}

	instanceTypes, err := provider.List(context.Background(), nodeClass)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	for _, it := range instanceTypes {
		fmt.Printf("InstanceType: %s\n", it.Name)
		fmt.Printf("  CPU: %v\n", it.Capacity["cpu"])
		fmt.Printf("  Memory: %v\n", it.Capacity["memory"])
		fmt.Printf("  Pods: %v\n", it.Capacity["pods"])
		fmt.Printf("  Offerings: %d\n", len(it.Offerings))
	}
}
