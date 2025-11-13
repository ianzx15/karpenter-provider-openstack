package operator

import (
	"context"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/operator"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instancetype"
)

func init() {

	karpv1.NormalizedLabels = lo.Assign(karpv1.NormalizedLabels, map[string]string{})
}

type Operator struct {
	*operator.Operator

	InstanceTypeProvider instancetype.Provider
	InstanceProvider     instance.Provider
}

func NewOperator(ctx context.Context, op *operator.Operator) (context.Context, *Operator) {
	logger := log.FromContext(ctx)

	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		logger.Error(fmt.Errorf("CLUSTER_NAME not set"), "missing cluster name")
		os.Exit(1)
	}

	// 1. Autenticar no OpenStack
	logger.Info("Authenticating with OpenStack using environment variables")
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		logger.Error(err, "failed to read OpenStack auth options from environment")
		os.Exit(1)
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		logger.Error(err, "failed to authenticate OpenStack client")
		os.Exit(1)
	}

	// 2. Criar o Cliente de Computação (Compute Service Client)
	region := os.Getenv("OS_REGION_NAME")
	if region == "" {
		err := fmt.Errorf("OS_REGION_NAME must be set in environment")
		logger.Error(err, "failed to get region")
		os.Exit(1)
	}

	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: region,
	})
	if err != nil {
		logger.Error(err, "failed to create OpenStack Compute v2 client")
		os.Exit(1)
	}
	logger.Info("OpenStack client created successfully", "region", region)

	// 3. Inicializar Provedores Específicos
	instanceTypeProvider, err := instancetype.NewProvider(ctx, computeClient)
	if err != nil {
		logger.Error(err, "failed to create instance type provider")
		os.Exit(1)
	}

	instanceProvider := instance.NewProvider(computeClient, clusterName)
	lo.Must0(v1openstack.AddToScheme(op.Manager.GetScheme()))
	// 4. Retornar o Operador estendido
	return ctx, &Operator{
		Operator:             op,
		InstanceTypeProvider: instanceTypeProvider,
		InstanceProvider:     instanceProvider,
	}
}
