package cloudprovider

import (
	"context"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/joho/godotenv"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
)

// --------------------------------------------------------------------
// ✅ CLONE DO createRealComputeClient (igual ao seu integration test)
// --------------------------------------------------------------------
func createRealComputeClient(t *testing.T) *gophercloud.ServiceClient {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		t.Fatalf("Falha ao ler opções de auth do ambiente: %v", err)
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		t.Fatalf("Falha ao autenticar cliente: %v", err)
	}

	authResult := provider.GetAuthResult()
	if authResult == nil {
		t.Fatal("Falha ao obter resultado de autenticação (authResult is nil)")
	}

	_, ok := authResult.(tokens.CreateResult)
	if !ok {
		t.Fatalf("Resultado de autenticação não é do tipo esperado (tokens.CreateResult)")
	}

	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		t.Fatalf("Falha ao criar cliente de Compute V2: %v", err)
	}

	return computeClient
}

// --------------------------------------------------------------------
// ✅ MOCKS
// --------------------------------------------------------------------
type mockInstanceTypeProvider struct {
	listFn func(ctx context.Context, nc *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error)
}

func (m *mockInstanceTypeProvider) List(ctx context.Context, nc *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
	return m.listFn(ctx, nc)
}

type mockInstanceProvider struct {
	createFn func(ctx context.Context, nc *v1openstack.OpenStackNodeClass, claim *karpv1.NodeClaim, it []*cloudprovider.InstanceType) (*instance.Instance, error)
}

func (m *mockInstanceProvider) Create(ctx context.Context, nc *v1openstack.OpenStackNodeClass, claim *karpv1.NodeClaim, it []*cloudprovider.InstanceType) (*instance.Instance, error) {
	return m.createFn(ctx, nc, claim, it)
}

// --------------------------------------------------------------------
// ✅ TEST Create()
// --------------------------------------------------------------------
func TestCloudProviderCreate(t *testing.T) {
	t.Log("Carregando .env...")
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Falha ao carregar .env: %v", err)
	}

	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Pulando teste: defina RUN_INTEGRATION_TESTS=1 para usar OpenStack real.")
	}

	// Cria cliente REAL do OpenStack (igual ao integration test)
	realComputeClient := createRealComputeClient(t)

	ctx := context.Background()

	// --------------------------------------------
	// ✅ NodeClass simulada
	// --------------------------------------------
	nodeClass := &v1openstack.OpenStackNodeClass{
		Spec: v1openstack.OpenStackNodeClassSpec{
			Networks: []string{"8e9133dd-0907-42f2-866d-c7ad2af7eb9c"},
			UserData: "echo hello",
			ImageSelectorTerms: []v1openstack.OpenStackImageSelectorTerm{
				{ID: "62dee28f-987d-40f5-a308-051d59991da8"},
			},
		},
	}

	nodeClaim := &karpv1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mock-cloudprovider-test",
		},
		Spec: karpv1.NodeClaimSpec{
			Requirements: []karpv1.NodeSelectorRequirementWithMinValues{
				{
					NodeSelectorRequirement: corev1.NodeSelectorRequirement{
						Key:      "instance-type",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"general.small"},
						
					},
					// MinValues é opcional, pode deixar zero
				},
			},
		},
	}

	// --------------------------------------------
	// ✅ InstanceType mockado
	// --------------------------------------------
	it := &cloudprovider.InstanceType{
		Name: "general.small",
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement("instance-type", "In", "general.small"),
		),
		Capacity: map[corev1.ResourceName]resource.Quantity{
			"cpu":    resourceMust("2"),
			"memory": resourceMust("4Gi"),
		},
	}

	mockITProvider := &mockInstanceTypeProvider{
		listFn: func(ctx context.Context, nc *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
			return []*cloudprovider.InstanceType{it}, nil
		},
	}

	// --------------------------------------------
	// ✅ InstanceProvider que MINIMAMENTE usa compute real
	// --------------------------------------------
	mockInstProvider := &mockInstanceProvider{
		createFn: func(ctx context.Context, nc *v1openstack.OpenStackNodeClass, claim *karpv1.NodeClaim, types []*cloudprovider.InstanceType) (*instance.Instance, error) {

			listOpts := servers.ListOpts{}

			_, err := servers.List(realComputeClient, listOpts).AllPages()
			if err != nil {
				t.Fatalf("Falha ao listar instâncias reais do OpenStack: %v", err)
			}

			t.Logf("OpenStack compute real operante (GET executado).")

			return &instance.Instance{
				InstanceID: "fake-id-123",
				Name:       "fake-name",
				Type:       "general.small",
				ImageID:    nc.Spec.ImageSelectorTerms[0].ID,
				Status:     "BUILD",
			}, nil
		},
	}

	// --------------------------------------------
	// ✅ Injetar CloudProvider com mocks
	// --------------------------------------------
	cp := &CloudProvider{
		instanceProvider:     mockInstProvider,
		instanceTypeProvider: mockITProvider,
	}

	// --------------------------------------------
	// ✅ Executa o Create real
	// --------------------------------------------
	result, err := cp.Create(ctx, nodeClaim)
	if err != nil {
		t.Fatalf("Create falhou: %v", err)
	}

	if result.Status.ProviderID == "" {
		t.Errorf("ProviderID não foi preenchido")
	}

	if result.Status.ImageID != nodeClass.Spec.ImageSelectorTerms[0].ID {
		t.Errorf("Imagem incorreta: %s", result.Status.ImageID)
	}

	t.Logf("NodeClaim criado com sucesso: %#v", result)
}

// Utilitário
func resourceMust(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
