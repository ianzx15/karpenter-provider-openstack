package cloudprovider

import (
	"context"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instancetype"
	"github.com/joho/godotenv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

func TestCloudProviderCreate_Integration(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Erro ao carregar .env: %v", err)
	}
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Pulando teste de integração. Set RUN_INTEGRATION_TESTS=1 para executar.")
	}

	//Valores requeridos pelo kubernetes
	const (
		nodeClassName = "test-node-class"
		nodeClaimName = "Karpenter-integ-test"
		imageID       = "62dee28f-987d-40f5-a308-051d59991da8"
		instanceID    = "mock-instance-uuid-456"

		flavorSmall = "7441c7d9-2648-4a33-907e-4d28c2270da3"

		flavorLarge  = "be9875d8-f22b-426e-91e1-79f04c705c09"
		flavorMedium = "69495bdc-cc5a-4596-9b0a-e2c30956df46"
	)

	flavorsList := []*flavors.Flavor{
		{
			Name:  flavorLarge,
			VCPUs: 4,
			RAM:   8192,
			ID:    "flavor-id-large",
		},
		// O flavor CORRETO
		{
			Name:  flavorSmall,
			VCPUs: 1,
			RAM:   2048,
			ID:    "flavor-id-small",
		},
		{
			Name:  flavorMedium,
			VCPUs: 4,
			RAM:   4096,
			ID:    "flavor-id-medium",
		},
	}

	ctx := context.Background()

	// Objeto NodeClass que esperamos que o KubeClient encontre
	nodeClass := &v1openstack.OpenStackNodeClass{
		ObjectMeta: metav1.ObjectMeta{Name: nodeClassName},
		Spec: v1openstack.OpenStackNodeClassSpec{
			ImageSelectorTerms: []v1openstack.OpenStackImageSelectorTerm{{ID: imageID}},
			UserData:           "#!/bin/bash",
		},
	}

	//Objeto NodeClaim que será passado para a função Create
	nodeClaim := &karpv1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: nodeClaimName},
		Spec: karpv1.NodeClaimSpec{
			NodeClassRef: &karpv1.NodeClassReference{
				Name: nodeClassName,
			},
			Requirements: []karpv1.NodeSelectorRequirementWithMinValues{
				{
					NodeSelectorRequirement: corev1.NodeSelectorRequirement{
						Key:      corev1.LabelInstanceTypeStable,
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{flavorMedium},
					},
				},
			},
			Resources: karpv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			},
		},
	}

	//Configurar provedores reais
	realComputeClient := createRealComputeClient(t)

	realITProvider := &instancetype.DefaultProvider{
		InstanceTypesInfo: flavorsList,
	}

	realInstanceProvider := instance.NewProvider(realComputeClient, "test-cluster")

	// Configurar o fake KubeClient
	scheme := runtime.NewScheme()
	require.NoError(t, v1openstack.AddToScheme(scheme))
	require.NoError(t, v1openstack.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodeClass).
		Build()

	// Instanciar o CloudProvider

	cp := &CloudProvider{
		kubeClient:           fakeClient,
		instanceTypeProvider: realITProvider,
		instanceProvider:     realInstanceProvider,
	}

	t.Log("Executando cp.Create")
	createdNodeClaim, err := cp.Create(ctx, nodeClaim)

	// --- Assert (Verificação) ---
	require.NoError(t, err, "A função Create não deve retornar erro")
	require.NotNil(t, createdNodeClaim, "O NodeClaim retornado não deve ser nulo")

	// Verificar Status (agora com dados reais)
	assert.NotEmpty(t, createdNodeClaim.Status.ProviderID)
	assert.Contains(t, createdNodeClaim.Status.ProviderID, "openstack://")
	assert.Equal(t, imageID, createdNodeClaim.Status.ImageID)

	// Verificar Labels
	assert.Equal(t, flavorSmall, createdNodeClaim.Labels[corev1.LabelInstanceTypeStable])
	assert.Equal(t, "amd64", createdNodeClaim.Labels[corev1.LabelArchStable]) // Assumindo amd64
	assert.Equal(t, "linux", createdNodeClaim.Labels[corev1.LabelOSStable])
	assert.Equal(t, flavorSmall, createdNodeClaim.Labels["instance-type"])

	// Verificar Capacity
	expectedCPU := resource.MustParse("2") // Baseado no 'general.small'
	actualCPU := createdNodeClaim.Status.Capacity[corev1.ResourceCPU]
	assert.Zerof(t, expectedCPU.Cmp(actualCPU), "CPU capacity mismatch: expected %s, got %s", expectedCPU.String(), actualCPU.String())

	expectedMem := resource.MustParse("4Gi") // Baseado no 'general.small'
	actualMem := createdNodeClaim.Status.Capacity[corev1.ResourceMemory]
	assert.Zerof(t, expectedMem.Cmp(actualMem), "Memory capacity mismatch: expected %s, got %s", expectedMem.String(), actualMem.String())

	t.Logf("Teste de integração do CloudProvider concluído com sucesso. ProviderID: %s", createdNodeClaim.Status.ProviderID)

}
