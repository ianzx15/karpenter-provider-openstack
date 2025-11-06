package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instancetype"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"

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

func TestCloudProviderCreate(t *testing.T) {

	//Valores requeridos pelo kubernetes
	const (
		nodeClassName = "test-node-class"
		nodeClaimName = "test-nodeclaim"
		imageID       = "test-image-id-123"
		instanceID    = "mock-instance-uuid-456"

		flavorSmall = "general.small"

		flavorTiny   = "general.tiny"
		flavorMedium = "general.medium"
	)

	flavorsList := []flavors.Flavor{
		{
			Name:  flavorTiny,
			VCPUs: 1,
			RAM:   2048,
			ID:    "flavor-id-tiny",
		},
		// O flavor CORRETO
		{
			Name:  flavorSmall,
			VCPUs: 2,
			RAM:   4096,
			ID:    "flavor-id-small",
		},
		{
			Name:  flavorMedium,
			VCPUs: 4,
			RAM:   8192,
			ID:    "flavor-id-medium",
		},
	}

	ctx := context.Background()

	// 1. Objeto NodeClass que esperamos que o KubeClient encontre
	nodeClass := &v1openstack.OpenStackNodeClass{
		ObjectMeta: metav1.ObjectMeta{Name: nodeClassName},
		Spec: v1openstack.OpenStackNodeClassSpec{
			ImageSelectorTerms: []v1openstack.OpenStackImageSelectorTerm{{ID: imageID}},
		},
	}

	realITProvider := &instancetype.DefaultProvider{
		InstanceTypesInfo: flavorsList,
	}

	realITProvider.List(ctx, nodeClass)

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
						Values:   []string{flavorSmall},
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

	// Instância que o mockInstanceProvider deve retornar
	returnedInstance := &instance.Instance{
		Name:       fmt.Sprintf("karpenter-%s", nodeClaimName),
		Type:       flavorSmall, // Importante: Type deve bater com o Name do InstanceType
		ImageID:    imageID,
		InstanceID: instanceID,
		Status:     "BUILD",
	}

	// Configurar o fake KubeClient
	scheme := runtime.NewScheme()
	require.NoError(t, v1openstack.AddToScheme(scheme))
	require.NoError(t, v1openstack.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodeClass). // Pré-carrega o NodeClass no cliente falso
		Build()

	// Configurar o mockInstanceProvider
	mockIProvider := &mockInstanceProvider{
		CreateFunc: func(ctx context.Context, nc *v1openstack.OpenStackNodeClass, n *karpv1.NodeClaim, its []*cloudprovider.InstanceType) (*instance.Instance, error) {
			// Verifica se os argumentos corretos foram passados
			assert.Equal(t, nodeClassName, nc.Name)
			assert.Equal(t, nodeClaimName, n.Name)
			require.Len(t, its, 1)
			assert.Equal(t, flavorSmall, its[0].Name)

			return returnedInstance, nil
		},
	}
	// Instanciar o CloudProvider com os mocks

	cp := &CloudProvider{
		kubeClient:           fakeClient,
		instanceTypeProvider: realITProvider,
		instanceProvider:     mockIProvider,
	}

	createdNodeClaim, err := cp.Create(ctx, nodeClaim)

	// --- Assert (Verificação) ---
	require.NoError(t, err, "A função Create não deve retornar erro")
	require.NotNil(t, createdNodeClaim, "O NodeClaim retornado não deve ser nulo")

	// Verificar Status
	expectedProviderID := fmt.Sprintf("openstack://%s/%s", instanceID, returnedInstance.Name)
	assert.Equal(t, expectedProviderID, createdNodeClaim.Status.ProviderID)
	assert.Equal(t, imageID, createdNodeClaim.Status.ImageID)

	// Verificar Labels
	assert.Equal(t, flavorSmall, createdNodeClaim.Labels[corev1.LabelInstanceTypeStable])
	assert.Equal(t, "amd64", createdNodeClaim.Labels[corev1.LabelArchStable])
	assert.Equal(t, "linux", createdNodeClaim.Labels[corev1.LabelOSStable])
	assert.Equal(t, flavorSmall, createdNodeClaim.Labels["instance-type"])

	// Verificar Capacity
	expectedCPU := resource.MustParse("2")
	actualCPU := createdNodeClaim.Status.Capacity[corev1.ResourceCPU]
	assert.Zerof(t, expectedCPU.Cmp(actualCPU), "CPU capacity mismatch: expected %s, got %s", expectedCPU.String(), actualCPU.String())

	expectedMem := resource.MustParse("4Gi")
	actualMem := createdNodeClaim.Status.Capacity[corev1.ResourceMemory]
	assert.Zerof(t, expectedMem.Cmp(actualMem), "Memory capacity mismatch: expected %s, got %s", expectedMem.String(), actualMem.String())
	assert.Equal(t, flavorSmall, createdNodeClaim.Labels[corev1.LabelInstanceTypeStable])
	// Debug
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

}
