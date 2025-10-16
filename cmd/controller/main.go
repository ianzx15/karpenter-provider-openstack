package main

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/cloudprovider"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instancetype"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1alpha1"
	"sigs.k8s.io/karpenter/pkg/events"
)

func main() {
	ctx := context.Background()

	kubeClient := fake.NewClientBuilder().Build()

	recorder := events.NewRecorder(nil, "openstack-test")

	instanceTypeProvider := instancetype.NewProvider() 
	instanceProvider := instance.NewProvider()         

	openstackProvider := cloudprovider.New(kubeClient, recorder, instanceTypeProvider, instanceProvider)

	nodeClaim := &karpv1.NodeClaim{
		Spec: karpv1.NodeClaimSpec{
			NodeClassRef: &karpv1.NodeClassRef{
				Name: "openstack-default",
			},
		},
	}

	newNodeClaim, err := openstackProvider.Create(ctx, nodeClaim)
	if err != nil {
		fmt.Println("Erro ao criar instância:", err)
		return
	}

	fmt.Println("Instância criada com ProviderID:", newNodeClaim.Status.ProviderID)
}
