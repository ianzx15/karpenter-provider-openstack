package main

import (
    "flag"
    "os"
    "context"
    "log"

    "github.com/ianzx15/karpenter-provider-openstack/pkg/openstack"
    "github.com/ianzx15/karpenter-provider-openstack/pkg/provider/openstack"
    "sigs.k8s.io/karpenter/pkg/cloudprovider"
    "sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
    var (
        imageID	= flag.String("image-id", "", "OpenStack Image ID to use for nodes")
        flavorID	= flag.String("flavor-id", "", "OpenStack Flavor ID to use for nodes")
        networkIDs	= flag.String("network-ids", "", "Comma separated network IDs")
        zone	= flag.String("zone", "", "OpenStack availability zone")
    )
    flag.Parse()

    cfg := openstack.Config{
        ImageID:   *imageID,
        FlavorID:  *flavorID,
        NetworkIDs: []string{*networkIDs}, 
        Zone:      *zone,
    }

    client, err := openstack.NewClient(/* credenciais do .env*/)
    if err != nil {
        log.Fatalf("failed to create openstack client: %v", err)
    }

    provider := openstack.NewCloudProvider(client, cfg)

    mgr, err := manager.New(manager.Options{
    })
    if err != nil {
        log.Fatalf("failed to create manager: %v", err)
    }

    cloudprovider.Register(provider)

    if err := mgr.Start(context.Background()); err != nil {
        log.Fatalf("manager exited: %v", err)
    }
}
