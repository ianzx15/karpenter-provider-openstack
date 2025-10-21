package openstack

import (
	"context"
	"fmt"
	"time"

	// IMPORT ADICIONADO para a interface Client
	"github.com/ianzx15/karpenter-provider-openstack/pkg/openstack"

	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	// API v1 (como você já tem)
	v1 "sigs.k8s.io/karpenter/pkg/apis/v1" 

	"github.com/ianzx15/karpenter-provider-openstack/pkg/utils"
)

type CloudProvider struct {
	// CORRIGIDO: O tipo é openstack.Client
	client openstack.Client 
	cfg    Config
}

// CORRIGIDO: O tipo é openstack.Client
func NewCloudProvider(client openstack.Client, cfg Config) *CloudProvider {
	return &CloudProvider{client: client, cfg: cfg}
}

func (p *CloudProvider) Create(ctx context.Context, nc *v1.NodeClaim) (*v1.NodeClaim, error) {
	// CORRIGIDO: O campo agora é nc.Spec.Kubelet, não KubeletConfiguration
	userdata := utils.GenerateUserData(nc.Spec.Kubelet.ClusterName, nc.Spec.Kubelet.ClusterEndpoint)
	
	serverID, err := p.client.CreateServer(ctx, "karpenter-"+nc.Name, p.cfg.ImageID, p.cfg.FlavorID, userdata, p.cfg.NetworkIDs, nil)
	if err != nil {
		// CORRIGIDO: O tipo de erro mudou para NewInsufficientCapacityError
		return nil, cloudprovider.NewInsufficientCapacityError(fmt.Errorf("OpenStack Create Failed: %w", err))
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

// CORRIGIDO: A assinatura da interface mudou.
// Agora recebe um *v1.NodePool, não KubeletConfiguration.
func (p *CloudProvider) GetInstanceTypes(ctx context.Context, nodePool *v1.NodePool) ([]*cloudprovider.InstanceType, error) {
	return BuildInstanceTypes(p.cfg)
}