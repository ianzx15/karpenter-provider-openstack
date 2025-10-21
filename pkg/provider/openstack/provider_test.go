// pkg/openstack/provider_test.go
package openstack

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCloudProvider_Create_HappyPath testa o cenário ideal de criação.
func TestCloudProvider_Create_HappyPath(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{t: t}
	cfg := Config{
		ImageID:  "test-image",
		FlavorID: "test-flavor",
	}
	provider := NewCloudProvider(mockClient, cfg)

	nodeClaim := &v1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Spec: v1.NodeClaimSpec{
			KubeletConfiguration: &v1.KubeletConfiguration{
				ClusterName:     "test-cluster",
				ClusterEndpoint: "https://test-api.com",
			},
		},
	}

	expectedServerID := "server-abc-123"
	expectedProviderID := fmt.Sprintf("openstack:///%s", expectedServerID)

	// Configura o comportamento do Mock
	mockClient.CreateServerFunc = func(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error) {
		assert.Contains(t, name, "karpenter-test-node")
		assert.Equal(t, cfg.ImageID, imageID)
		assert.Equal(t, cfg.FlavorID, flavorID)
		// Você pode adicionar mais asserts no userdata aqui
		return expectedServerID, nil
	}

	mockClient.GetServerFunc = func(ctx context.Context, id string) (ServerInfo, error) {
		assert.Equal(t, expectedServerID, id)
		return ServerInfo{
			ID:     expectedServerID,
			Status: "ACTIVE", // Retorna ACTIVE na primeira tentativa
		}, nil
	}

	// Executa a função
	// Reduz o tempo de poll para o teste não demorar 5 minutos
    // ATENÇÃO: Seu código atual tem um poll hardcoded de 5s * 60.
    // Para testar rapidamente, você precisaria refatorar o provider.Create
    // para aceitar um timeout/intervalo ou aceitar que o teste vai demorar.
    // Assumindo que o mock retorna "ACTIVE" de primeira, o loop sairá.
	
	resultNC, err := provider.Create(ctx, nodeClaim)

	// Verifica os resultados
	require.NoError(t, err)
	assert.Equal(t, expectedProviderID, resultNC.Status.ProviderID)
	assert.Equal(t, 1, mockClient.CreateServerCalls)
	assert.Equal(t, 1, mockClient.GetServerCalls)
}

// TestCloudProvider_Create_Polling testa o cenário onde o servidor demora a ficar ATIVO.
func TestCloudProvider_Create_Polling(t *testing.T) {
    // ... setup similar ao anterior ...
	mockClient := &MockClient{t: t}
	provider := NewCloudProvider(mockClient, Config{})
	nodeClaim := &v1.NodeClaim{ObjectMeta: metav1.ObjectMeta{Name: "poll-node"}} // simplificado

	expectedServerID := "server-poll-456"

	mockClient.CreateServerFunc = func(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error) {
		return expectedServerID, nil
	}

    // Simula o poll
	callCount := 0
	mockClient.GetServerFunc = func(ctx context.Context, id string) (ServerInfo, error) {
		callCount++
		if callCount < 3 {
			// Nas primeiras 2 chamadas, retorna BUILD
			return ServerInfo{ID: id, Status: "BUILD"}, nil
		}
		// Na 3ª chamada, retorna ACTIVE
		return ServerInfo{ID: id, Status: "ACTIVE"}, nil
	}
	
	// ATENÇÃO: Este teste ainda depende do `time.Sleep(5 * time.Second)`
    // no seu provider.Create. Ele levará ~10 segundos (2 polls de 5s).
	
	resultNC, err := provider.Create(ctx, nodeClaim)

	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("openstack:///%s", expectedServerID), resultNC.Status.ProviderID)
	assert.Equal(t, 1, mockClient.CreateServerCalls, "CreateServer deveria ser chamado 1 vez")
	assert.Equal(t, 3, mockClient.GetServerCalls, "GetServer deveria ser chamado 3 vezes")
}

// TestCloudProvider_Delete testa a deleção.
func TestCloudProvider_Delete(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{t: t}
	provider := NewCloudProvider(mockClient, Config{})

	serverID := "server-to-delete-789"
	providerID := fmt.Sprintf("openstack:///%s", serverID)

	// Configura o Mock
	mockClient.DeleteServerFunc = func(ctx context.Context, id string) error {
		assert.Equal(t, serverID, id, "O ID extraído do providerID está incorreto")
		return nil // Sucesso
	}

	// Executa
	err := provider.Delete(ctx, providerID)

	// Verifica
	require.NoError(t, err)
	assert.Equal(t, 1, mockClient.DeleteServerCalls)
}

// TestCloudProvider_Delete_BadID testa um providerID mal formatado.
func TestCloudProvider_Delete_BadID(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{t: t}
	provider := NewCloudProvider(mockClient, Config{})

	// Não configura o mockClient.DeleteServerFunc, pois não deve ser chamado
	mockClient.DeleteServerFunc = func(ctx context.Context, id string) error {
		t.Error("DeleteServer não deveria ter sido chamado com um ID inválido")
		return nil
	}
	
	err := provider.Delete(ctx, "id-formato-invalido")

	require.Error(t, err, "Deveria falhar ao parsear o ID")
	assert.Equal(t, 0, mockClient.DeleteServerCalls)
}