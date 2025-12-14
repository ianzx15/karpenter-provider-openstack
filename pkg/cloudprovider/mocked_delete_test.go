package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	CreateFunc func(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*instance.Instance, error)
	DeleteFunc func(ctx context.Context, providerID string) error // Adicionado o retorno de erro para o DeleteFunc
}

func (m *mockProvider) Create(ctx context.Context, nodeClass *v1openstack.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*instance.Instance, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, nodeClass, nodeClaim, instanceTypes)
	}
	return nil, fmt.Errorf("CreateFunc não implementado")
}

func (m *mockProvider) Delete(ctx context.Context, providerID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, providerID)
	}
	return nil
}

func TestCloudProviderDelete(t *testing.T) {
	const (
		nodeClaimName = "delete-test-nodeclaim"
		serverID      = "f22b426e-91e1-79f04c705c09"
		providerID    = "openstack:///" + serverID
	)

	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		nodeClaim := &karpv1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: nodeClaimName},
			Status: karpv1.NodeClaimStatus{
				ProviderID: providerID,
			},
		}

		deleteCalledWith := ""
		mockIProvider := &mockProvider{
			DeleteFunc: func(_ context.Context, id string) error {
				deleteCalledWith = id
				return nil
			},
		}

		cp := &CloudProvider{
			instanceProvider: mockIProvider,
		}

		err := cp.Delete(ctx, nodeClaim)

		require.NoError(t, err, "A exclusão não deve retornar erro")
		assert.Equal(t, serverID, deleteCalledWith, "A instância deve ser excluída usando o Server ID extraído, não o Provider ID completo.")
	})

	t.Run("missing providerID", func(t *testing.T) {
		nodeClaim := &karpv1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: nodeClaimName},
			Status:     karpv1.NodeClaimStatus{ProviderID: ""},
		}

		cp := &CloudProvider{
			instanceProvider: &mockProvider{},
		}

		err := cp.Delete(ctx, nodeClaim)

		require.Error(t, err, "A exclusão deve retornar erro se ProviderID estiver faltando")
		assert.Contains(t, err.Error(), "missing ProviderID", "Mensagem de erro incorreta para ProviderID ausente")
	})

	t.Run("invalid providerID format", func(t *testing.T) {

		nodeClaim := &karpv1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: nodeClaimName},
			Status: karpv1.NodeClaimStatus{
				ProviderID: "invalid-format-id",
			},
		}

		cp := &CloudProvider{
			instanceProvider: &mockProvider{},
		}

		err := cp.Delete(ctx, nodeClaim)

		require.Error(t, err, "A exclusão deve retornar erro se ProviderID tiver um formato incorreto")
		assert.Contains(t, err.Error(), "unexpected providerID format", "Mensagem de erro incorreta para formato de ProviderID inválido")	})
	}
