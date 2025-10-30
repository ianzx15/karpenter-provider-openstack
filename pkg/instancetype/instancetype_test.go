package instancetype_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instancetype"
)

func TestListInstanceTypes(t *testing.T) {
	mockFlavors := []*flavors.Flavor{
		{
			Name:  "small",
			VCPUs: 2,
			RAM:   4096,
		},
		{
			Name:  "medium",
			VCPUs: 4,
			RAM:   8192,
		},
	}

	provider := &instancetype.DefaultProvider{
		InstanceTypesInfo: mockFlavors,
	}

	nodeClass := &v1openstack.OSNodeClass{
		Spec: v1openstack.OSNodeClassSpec{
			Flavor: "small",
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
