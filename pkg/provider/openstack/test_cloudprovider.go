package openstack_test

import (
    "context"
    "testing"

    osprovider "github.com/yourorg/karpenter-provider-openstack/pkg/provider/openstack"
    "github.com/yourorg/karpenter-provider-openstack/pkg/openstack"
)

func TestCreateNode(t *testing.T) {
    mock := &openstack.MockClient{}
    cfg := osprovider.Config{ImageID: "img", FlavorID: "flv", NetworkIDs: []string{"net"}}
    provider := osprovider.NewCloudProvider(mock, cfg)

    node := &v1.NodeClaim{}
    created, err := provider.Create(context.Background(), node)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if created.Status.ProviderID == "" {
        t.Fatalf("expected ProviderID to be set")
    }
}
