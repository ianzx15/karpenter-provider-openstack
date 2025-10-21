package openstack

import (
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    "sigs.k8s.io/karpenter/pkg/cloudprovider"
)

func BuildInstanceTypes(cfg Config) ([]cloudprovider.InstanceType, error) {
    it := cloudprovider.InstanceType{
        Name: "openstack-mvp",
        Requirements: map[string][]string{
            "topology.kubernetes.io/zone": {cfg.Zone},
        },
        Resources: corev1.ResourceList{
            corev1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
            corev1.ResourceMemory: *resource.NewQuantity(4*1024*1024*1024, resource.BinarySI),
        },
        Offerings: []cloudprovider.Offering{
            {Zone: cfg.Zone, CapacityType: "on-demand"},
        },
    }
    return []cloudprovider.InstanceType{it}, nil
}
