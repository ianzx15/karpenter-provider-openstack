package instance

import (
    "context"
    "fmt"
    "log"

    "github.com/gophercloud/gophercloud"
    "github.com/gophercloud/gophercloud/openstack"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type DefaultProvider struct {
    novaClient *gophercloud.ServiceClient
}

func NewProviderOpenStack() (*DefaultProvider, error) {
    opts, err := openstack.AuthOptionsFromEnv()
    if err != nil {
        return nil, fmt.Errorf("failed to get auth options: %w", err)
    }

    provider, err := openstack.AuthenticatedClient(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to authenticate: %w", err)
    }

    nova, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
        Region: "Mudar para testes",
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create compute client: %w", err)
    }

    return &DefaultProvider{
        novaClient: nova,
    }, nil
}

func (p *DefaultProvider) Create(ctx context.Context, name, flavorID, imageID, networkID string) (string, error) {
    createOpts := servers.CreateOpts{
        Name:      name,
        FlavorRef: flavorID,
        ImageRef:  imageID,
        Networks:  []servers.Network{{UUID: networkID}},
        UserData: []byte(`#cloud-config
runcmd:
 - echo "Hello cloud init" > /root/hello.txt
`),
    }

    server, err := servers.Create(p.novaClient, createOpts).Extract()
    if err != nil {
        return "", fmt.Errorf("failed to create instance: %w", err)
    }

    log.Printf("Created instance %s (%s)\n", server.Name, server.ID)
    return server.ID, nil
}
