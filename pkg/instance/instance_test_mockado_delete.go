package instance

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
)

// Requisito: As estruturas MockRoundTripper e createMockServiceClient devem estar presentes.

// MockRoundTripper simula a resposta HTTP da API do OpenStack.
type MockRoundTripper struct {
	Response *http.Response
	Err      error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

// createMockServiceClient cria um ServiceClient mockado para OpenStack.
func createMockServiceClient(statusCode int, responseBody string, err error) *gophercloud.ServiceClient {
	resp := &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       http.NoBody,
		Header:     make(http.Header),
	}
	if responseBody != "" {
		resp.Body = io.NopCloser(strings.NewReader(responseBody))
	}

	return &gophercloud.ServiceClient{
		Endpoint:     "http://mock-openstack:8774/v2.1",
		ResourceBase: "http://mock-openstack:8774/v2.1/",
		Type:         "compute",
		ProviderClient: &gophercloud.ProviderClient{
			HTTPClient: http.Client{
				Transport: &MockRoundTripper{
					Response: resp,
					Err:      err,
				},
			},
		},
	}
}

func TestDeleteInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		providerID      string
		mockStatusCode  int
		mockBody        string
		expectError     bool
		isNotFoundError bool // Espera que o erro seja cloudprovider.NodeClaimNotFoundError
	}{
		{
			name:           "1. Sucesso na Exclusão (Status 204)",
			providerID:     "openstack:///mock-instance-id-success",
			mockStatusCode: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:            "2. Instância Não Encontrada (Status 404)",
			providerID:      "openstack:///mock-instance-id-404",
			mockStatusCode:  http.StatusNotFound,
			mockBody:        `{"itemNotFound": {"message": "Server not found", "code": 404}}`,
			expectError:     true,
			isNotFoundError: true, // Espera que seja traduzido para NodeClaimNotFoundError
		},
		{
			name:            "3. Erro na API (Status 400 Bad Request)",
			providerID:      "openstack:///mock-instance-id-400",
			mockStatusCode:  http.StatusBadRequest,
			mockBody:        `{"badRequest": {"message": "Invalid parameters", "code": 400}}`,
			expectError:     true,
			isNotFoundError: false,
		},
		{
			name:            "4. ProviderID com Formato Inválido",
			providerID:      "invalid-prefix/mock-instance-id",
			mockStatusCode:  http.StatusNoContent,
			expectError:     true, // Deve falhar no parse, antes de chamar o mock
			isNotFoundError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Configurar o Cliente Mockado
			mockClient := createMockServiceClient(tt.mockStatusCode, tt.mockBody, nil)

			// 2. Criar o Provedor com o Cliente Mockado
			provider := NewProvider(mockClient, "mock-cluster")
			defaultProvider := provider.(*DefaultProvider)

			// 3. Executar a função Delete
			err := defaultProvider.Delete(ctx, tt.providerID)

			// 4. Verificar o resultado
			if tt.expectError {
				if err == nil {
					t.Fatalf("Esperava um erro, mas recebi nil")
				}

				if tt.isNotFoundError {
					// Verifica se o erro é do tipo NodeClaimNotFoundError
					if !cloudprovider.IsNodeClaimNotFoundError(err) {
						t.Errorf("Esperava cloudprovider.NodeClaimNotFoundError, mas recebi %v", err)
					}
				} else if cloudprovider.IsNodeClaimNotFoundError(err) {
					// Se não esperava NodeClaimNotFoundError, mas recebeu.
					t.Errorf("Esperava um erro genérico, mas recebi cloudprovider.NodeClaimNotFoundError")
				}
			} else { // Não esperava erro
				if err != nil {
					t.Errorf("Não esperava erro, mas recebi: %v", err)
				}
			}
		})
	}
}
