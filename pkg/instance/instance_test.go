package instance

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
)

func TestCreateInstance(t *testing.T) {
	ctx := context.Background()

	nodeClass := &OpenStackNodeClass{
		Spec: OpenStackNodeClassSpec{
			ImageRef: "mock-image-id-456",
			Networks: []string{"net-uuid-1"},
			UserData: "#!/bin/bash\necho 'hello world'",
		},
	}
	nodeClaim := &karpv1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node-claim"},
	}

	instanceType := &cloudprovider.InstanceType{
		Name: "m1.large",
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement("instance-type", "In", "m1.large"),
		),
	}

	test := NewProvider(nil, "test-cluster")
	visu := test.(*DefaultProvider)
	fmt.Println("aquiiii:", visu.clusterName)

	instance, err := test.Create(ctx, nodeClass, nodeClaim, []*cloudprovider.InstanceType{instanceType})
	if err != nil {
		t.Fatalf("Falha ao criar instância: %v", err)
	}

	if instance.InstanceID != "mock-server-id-123" {
		t.Errorf("ID incorreto: esperado='mock-server-id-123', obtido='%s'", instance.InstanceID)
	}
	if instance.Type != "m1.large" {
		t.Errorf("Tipo incorreto: esperado='m1.large', obtido='%s'", instance.Type)
	}
	if instance.ImageID != "mock-image-id-456" {
		t.Errorf("ImageID incorreto: esperado='mock-image-id-456', obtido='%s'", instance.ImageID)
	}
	if instance.Status != "BUILD" {
		t.Errorf("Status incorreto: esperado='BUILD', obtido='%s'", instance.Status)
	}
}
