package instance

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
)

type Provider interface {
	Create(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*Instance, error)
}

type DefaultProvider struct {
	clusterName   string
	computeClient *gophercloud.ServiceClient
}

func NewProvider(client *gophercloud.ServiceClient, clusterName string) Provider {
	return &DefaultProvider{
		clusterName:   clusterName,
		computeClient: client,
	}
}

func (p *DefaultProvider) Create(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*Instance, error) {
	if len(instanceTypes) == 0 {
		return nil, fmt.Errorf("no instance types provided")
	}
	capacityType := karpv1.CapacityTypeOnDemand
	zone := "default-zone"

	var errs []error
	for _, instanceType := range instanceTypes {
		instanceName := fmt.Sprintf("karpenter-%s", nodeClaim.Name)

		createdOpts, err := p.buildInstanceOpts(ctx, nodeClaim, nodeClass, instanceType, zone, instanceName, capacityType)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to build instance options for %s: %w", instanceType.Name, err))
			continue
		}
		//Real creation
		server, err := servers.Create(p.computeClient, createdOpts).Extract()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create instance for %s: %w", instanceType.Name, err))
			continue
		}

		instance := &Instance{
			Name:       server.Name,
			Type:       server.Flavor["id"].(string),
			ImageID:    server.Image["id"].(string),
			Metadata:   server.Metadata,
			UserData:   createdOpts.UserData,
			InstanceID: server.ID,
			Status:     server.Status,
		}

		log.FromContext(ctx).Info("Creating instance OpenStack", "instanceName", instanceName, "flavor", instanceType.Name, "zone", zone)
		//Mocked creation
		// instance := &Instance{
		// 	Name:       createdOpts.Name,
		// 	Type:       createdOpts.FlavorRef,
		// 	ImageID:    createdOpts.ImageRef,
		// 	Metadata:   createdOpts.Metadata,
		// 	UserData:   createdOpts.UserData,
		// 	InstanceID: "mock-server-id-123",
		// 	Status:     "BUILD",
		// }

		fmt.Printf("Instance successfully created | instanceName=%s | providerID=%s | status=%s | UserData=%s | MetaData=%s | ImageId=%s | Type=%s\n",
			instance.Name, instance.InstanceID, instance.Status, string(instance.UserData), instance.Metadata, instance.ImageID, instance.Type)
		return instance, nil
	}

	return nil, fmt.Errorf("failed to create instance after trying all instance types: %w", fmt.Errorf("%v", errs))
}

func (p *DefaultProvider) buildInstanceOpts(ctx context.Context, nodeClaim *karpv1.NodeClaim, nodeClass *v1openstack.OpenStackNodeClass, instanceType *cloudprovider.InstanceType, zone, instanceName, capacityType string) (servers.CreateOpts, error) {
	imageID := nodeClass.Spec.ImageSelectorTerms[0].ID
	flavorName := instanceType.Name

	userData := []byte(nodeClass.Spec.UserData)

	return servers.CreateOpts{
		Name:      instanceName,
		FlavorRef: flavorName,
		ImageRef:  imageID,
		UserData:  userData,
	}, nil
}
