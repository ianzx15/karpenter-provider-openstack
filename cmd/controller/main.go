package main

import (
	"context"
	"fmt"
	"log"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/cloudprovider"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/providers/instancetype"

	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

func main() {
	ctx := context.Background()

	k8sConfig := config.GetConfigOrDie()
	mgr, err := manager.New(k8sConfig, manager.Options{})
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	instanceProvider, err := instance.NewProviderOpenStack()
	if err != nil {
		log.Fatalf("failed to initialize OpenStack instance provider: %v", err)
	}

	instanceTypeProvider := instancetype.NewProvider() 

	openstackProvider := cloudprovider.New(
		mgr.GetClient(),
		nil, 
		instanceTypeProvider,
		instanceProvider,
	)

	nodeClaim := &karpv1.NodeClaim{
		Spec: karpv1.NodeClaimSpec{
			NodeClassRef: &karpv1.NodeClassRef{
				Name: "openstack-small", 
			},
		},
	}

	newNodeClaim, err := openstackProvider.Create(ctx, nodeClaim)
	if err != nil {
		log.Fatalf("failed to create OpenStack instance: %v", err)
	}

	fmt.Printf("Instância criada com ProviderID: %s\n", newNodeClaim.Status.ProviderID)
}
