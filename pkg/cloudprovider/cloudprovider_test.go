package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockInstanceProvider é um mock para a interface instance.Provider
type mockInstanceProvider struct {
	CreateFunc func(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*instance.Instance, error)
}

func (m *mockInstanceProvider) Create(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*instance.Instance, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, nodeClass, nodeClaim, instanceTypes)
	}
	return nil, fmt.Errorf("CreateFunc não implementado")
}

// mockInstanceTypeProvider é um mock para a interface instancetype.Provider
type mockInstanceTypeProvider struct {
	ListFunc func(context.Context, *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error)
}

func (m *mockInstanceTypeProvider) List(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, nodeClass)
	}
	return nil, fmt.Errorf("ListFunc não implementado")
}

// TestCloudProviderCreate testa o "caminho feliz" da função Create
func TestCloudProviderCreate(t *testing.T) {
	// --- Arrange (Configuração) ---
	//Valores requeridos pelo kubernetes
	const (
		nodeClassName = "test-node-class"
		nodeClaimName = "test-nodeclaim"
		flavorName    = "general.small"
		imageID       = "test-image-id-123"
		instanceID    = "mock-instance-uuid-456"
	)

	ctx := context.Background()

	// 1. Objeto NodeClass que esperamos que o KubeClient encontre
	nodeClass := &v1openstack.OpenStackNodeClass{
		ObjectMeta: metav1.ObjectMeta{Name: nodeClassName},
		Spec: v1openstack.OpenStackNodeClassSpec{
			ImageSelectorTerms: []v1openstack.OpenStackImageSelectorTerm{{ID: imageID}},
		},
	}

	// 2. Objeto NodeClaim que será passado para a função Create
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
						Values:   []string{flavorName},
					},
				},
			},
			Resources: karpv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
			},
		},
	}

	// 3. Tipo de instância que o mockInstanceTypeProvider deve retornar
	testInstanceType := &cloudprovider.InstanceType{
		Name: flavorName,
		Capacity: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			corev1.ResourcePods:   resource.MustParse("110"),
		},
		Overhead: &cloudprovider.InstanceTypeOverhead{},
		Offerings: cloudprovider.Offerings{
			{
				Requirements: scheduling.NewRequirements(
					scheduling.NewRequirement(karpv1.CapacityTypeLabelKey, corev1.NodeSelectorOpIn, string(karpv1.CapacityTypeOnDemand)),
				),
				Available: true,
			},
		},
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement(corev1.LabelInstanceTypeStable, corev1.NodeSelectorOpIn, flavorName),
			scheduling.NewRequirement(corev1.LabelArchStable, corev1.NodeSelectorOpIn, "amd64"),
			scheduling.NewRequirement(corev1.LabelOSStable, corev1.NodeSelectorOpIn, "linux"),
		),
	}

	// 4. Instância que o mockInstanceProvider deve retornar
	returnedInstance := &instance.Instance{
		Name:       fmt.Sprintf("karpenter-%s", nodeClaimName),
		Type:       flavorName, // Importante: Type deve bater com o Name do InstanceType
		ImageID:    imageID,
		InstanceID: instanceID,
		Status:     "BUILD",
	}

	// 5. Configurar o fake KubeClient
	scheme := runtime.NewScheme()
	require.NoError(t, v1openstack.AddToScheme(scheme))
	require.NoError(t, v1openstack.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodeClass). // Pré-carrega o NodeClass no cliente falso
		Build()

	// 6. Configurar o mockInstanceTypeProvider
	mockITProvider := &mockInstanceTypeProvider{
		ListFunc: func(ctx context.Context, nc *v1openstack.OpenStackNodeClass) ([]*cloudprovider.InstanceType, error) {
			// Verifica se o NodeClass correto foi recebido
			assert.Equal(t, nodeClassName, nc.Name)
			return []*cloudprovider.InstanceType{testInstanceType}, nil
		},
	}

	// 7. Configurar o mockInstanceProvider
	mockIProvider := &mockInstanceProvider{
		CreateFunc: func(ctx context.Context, nc *v1openstack.OpenStackNodeClass, n *karpv1.NodeClaim, its []*cloudprovider.InstanceType) (*instance.Instance, error) {
			// Verifica se os argumentos corretos foram passados
			assert.Equal(t, nodeClassName, nc.Name)
			assert.Equal(t, nodeClaimName, n.Name)
			require.Len(t, its, 1) // Garante que o filtro de instancetype funcionou
			assert.Equal(t, flavorName, its[0].Name)

			return returnedInstance, nil
		},
	}

	// 8. Instanciar o CloudProvider com os mocks
	// (Veja a Nota 1 abaixo sobre por que instanciamos manualmente)
	cp := &CloudProvider{
		kubeClient:           fakeClient,
		instanceTypeProvider: mockITProvider,
		instanceProvider:     mockIProvider,
	}

	// --- Act (Execução) ---
	createdNodeClaim, err := cp.Create(ctx, nodeClaim)

	// --- Assert (Verificação) ---
	require.NoError(t, err, "A função Create não deve retornar erro")
	require.NotNil(t, createdNodeClaim, "O NodeClaim retornado não deve ser nulo")

	// Verificar Status
	expectedProviderID := fmt.Sprintf("openstack://%s/%s", instanceID, returnedInstance.Name)
	assert.Equal(t, expectedProviderID, createdNodeClaim.Status.ProviderID)
	assert.Equal(t, imageID, createdNodeClaim.Status.ImageID)

	// Verificar Labels (Veja a Nota 2 abaixo)
	assert.Equal(t, flavorName, createdNodeClaim.Labels[corev1.LabelInstanceTypeStable])
	assert.Equal(t, "amd64", createdNodeClaim.Labels[corev1.LabelArchStable])
	assert.Equal(t, "linux", createdNodeClaim.Labels[corev1.LabelOSStable])
	assert.Equal(t, flavorName, createdNodeClaim.Labels["instance-type"])

	// Verificar Capacity
	assert.Equal(t, resource.MustParse("2"), createdNodeClaim.Status.Capacity[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("4Gi"), createdNodeClaim.Status.Capacity[corev1.ResourceMemory])

	// --- Debug Print ---
	t.Log("==== DEBUG INFORMATION ====")

	t.Logf("NodeClass Name: %s", nodeClass.Name)
	t.Logf("NodeClass ImageSelectorTerms: %+v", nodeClass.Spec.ImageSelectorTerms)

	t.Logf("NodeClaim Name: %s", createdNodeClaim.Name)
	t.Logf("NodeClaim ProviderID: %s", createdNodeClaim.Status.ProviderID)
	t.Logf("NodeClaim ImageID: %s", createdNodeClaim.Status.ImageID)

	t.Log("Labels:")
	for k, v := range createdNodeClaim.Labels {
		t.Logf("  %s = %s", k, v)
	}

	t.Log("Capacity:")
	for k, v := range createdNodeClaim.Status.Capacity {
		t.Logf("  %s = %s", k.String(), v.String())
	}

	// Instance returned
	t.Logf("Returned Instance: {ID: %s, Name: %s, ImageID: %s, Type: %s, Status: %s}", 
		returnedInstance.InstanceID,
		returnedInstance.Name,
		returnedInstance.ImageID,
		returnedInstance.Type,
		returnedInstance.Status,
	)

	// Instance Type used
	t.Log("InstanceType:")
	t.Logf("  Name: %s", testInstanceType.Name)
	t.Logf("  Capacity: %+v", testInstanceType.Capacity)
	t.Logf("  Requirements: %+v", testInstanceType.Requirements)
	t.Logf("  Offerings: %+v", testInstanceType.Offerings)

	t.Log("==== END DEBUG ====")


}
