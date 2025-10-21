package openstack

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	
	// IMPORTS ADICIONADOS para a nova API
	"sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/scheduling"
)

func BuildInstanceTypes(cfg Config) ([]*cloudprovider.InstanceType, error) {
	it := &cloudprovider.InstanceType{
		Name: "openstack-mvp",
		// CORRIGIDO: Requirements agora usa esta estrutura
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement(corev1.LabelTopologyZone, corev1.NodeSelectorOpIn, cfg.Zone),
			scheduling.NewRequirement(v1.CapacityTypeLabel, corev1.NodeSelectorOpIn, v1.CapacityTypeOnDemand),
		),
		// CORRIGIDO: Resources foi renomeado para Capacity
		Capacity: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(4*1024*1024*1024, resource.BinarySI),
		},
		// CORRIGIDO: Offerings agora é uma slice de PONTEIROS
		Offerings: []*cloudprovider.Offering{
			{
				Zone:         cfg.Zone,
				CapacityType: v1.CapacityTypeOnDemand,
			},
		},
	}
	// CORRIGIDO: A função agora deve retornar []*cloudprovider.InstanceType
	return []*cloudprovider.InstanceType{it}, nil
}