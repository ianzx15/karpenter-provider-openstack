package openstack

import (
    "context"
    "fmt"
    "time"

    "sigs.k8s.io/karpenter/pkg/cloudprovider"
    v1 "sigs.k8s.io/karpenter/pkg/apis/v1"

    "github.com/ianzx15/karpenter-provider-openstack/pkg/utils"
)

type CloudProvider struct {
    client Client
    cfg    Config
}

func NewCloudProvider(client Client, cfg Config) *CloudProvider {
    return &CloudProvider{client: client, cfg: cfg}
}

func (p *CloudProvider) Create(ctx context.Context, nc *v1.NodeClaim) (*v1.NodeClaim, error) {
    userdata := utils.GenerateUserData(nc.Spec.KubeletConfiguration.ClusterName, nc.Spec.KubeletConfiguration.ClusterEndpoint)
    serverID, err := p.client.CreateServer(ctx, "karpenter-"+nc.Name, p.cfg.ImageID, p.cfg.FlavorID, userdata, p.cfg.NetworkIDs, nil)
    if err != nil {
        return nil, cloudprovider.NewCreateError(err, cloudprovider.InsufficientCapacity, "OpenStack Create Failed")
    }

    for i := 0; i < 60; i++ {
        srv, err := p.client.GetServer(ctx, serverID)
        if err == nil && srv.Status == "ACTIVE" {
            nc.Status.ProviderID = fmt.Sprintf("openstack:///%s", serverID)
            return nc, nil
        }
        time.Sleep(5 * time.Second)
    }

    return nil, fmt.Errorf("server did not become ACTIVE in time")
}

func (p *CloudProvider) Delete(ctx context.Context, providerID string) error {
    var id string
    _, err := fmt.Sscanf(providerID, "openstack:///%s", &id)
    if err != nil {
        return err
    }
    return p.client.DeleteServer(ctx, id)
}

func (p *CloudProvider) GetInstanceTypes(ctx context.Context, kc *v1.KubeletConfiguration) ([]cloudprovider.InstanceType, error) {
    return BuildInstanceTypes(p.cfg)
}
