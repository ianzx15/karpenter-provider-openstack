package openstack

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/gophercloud/gophercloud"
    "github.com/gophercloud/gophercloud/openstack"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
    "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
    "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

type Client interface {
    CreateServer(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error)
    GetServer(ctx context.Context, id string) (ServerInfo, error)
    DeleteServer(ctx context.Context, id string) error
}

type client struct {
    compute *gophercloud.ServiceClient
}

type ServerInfo struct {
    ID     string
    Name   string
    Status string
    IPs    []string
    CPU    int64
    Memory int64
}

func NewClient() (Client, error) {
    opts, err := openstack.AuthOptionsFromEnv()
    if err != nil {
        return nil, fmt.Errorf("failed to get auth options: %w", err)
    }

    provider, err := openstack.AuthenticatedClient(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to authenticate: %w", err)
    }

    compute, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
    if err != nil {
        return nil, fmt.Errorf("failed to create compute client: %w", err)
    }

    return &client{compute: compute}, nil
}

func (c *client) CreateServer(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error) {
    nets := make([]servers.Network, 0)
    for _, id := range networkIDs {
        nets = append(nets, servers.Network{UUID: id})
    }

    createOpts := servers.CreateOpts{
        Name:             name,
        FlavorRef:        flavorID,
        ImageRef:         imageID,
        Networks:         nets,
        UserData:         []byte(userdata),
        Metadata:         meta,
        SecurityGroups:   []string{"default"},
        AvailabilityZone: "",
    }

    server, err := servers.Create(c.compute, createOpts).Extract()
    if err != nil {
        return "", fmt.Errorf("failed to create server: %w", err)
    }

    log.Printf("created server %s (%s)", server.ID, server.Name)
    return server.ID, nil
}

func (c *client) GetServer(ctx context.Context, id string) (ServerInfo, error) {
    srv, err := servers.Get(c.compute, id).Extract()
    if err != nil {
        return ServerInfo{}, fmt.Errorf("failed to get server: %w", err)
    }

    var ips []string
    for _, addrs := range srv.Addresses {
        if arr, ok := addrs.([]interface{}); ok {
            for _, addr := range arr {
                m := addr.(map[string]interface{})
                if ip, ok := m["addr"].(string); ok {
                    ips = append(ips, ip)
                }
            }
        }
    }

    flavor, err := flavors.Get(c.compute, srv.Flavor["id"].(string)).Extract()
    if err != nil {
        log.Printf("warning: could not get flavor: %v", err)
    }

    return ServerInfo{
        ID:     srv.ID,
        Name:   srv.Name,
        Status: srv.Status,
        IPs:    ips,
        CPU:    int64(flavor.VCPUs),
        Memory: int64(flavor.RAM) * 1024 * 1024, // MB to bytes
    }, nil
}

func (c *client) DeleteServer(ctx context.Context, id string) error {
    return servers.Delete(c.compute, id).ExtractErr()
}
