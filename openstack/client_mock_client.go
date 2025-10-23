// pkg/openstack/client_mock_test.go
package openstack

import (
	"context"
	"testing"
)

// MockClient é uma implementação simulada da interface Client para testes.
type MockClient struct {
	t *testing.T // Para ajudar a relatar falhas

	// Campos para controlar o comportamento do mock
	CreateServerFunc func(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error)
	GetServerFunc    func(ctx context.Context, id string) (ServerInfo, error)
	DeleteServerFunc func(ctx context.Context, id string) error

	// Contadores de chamadas (opcional, mas útil)
	CreateServerCalls int
	GetServerCalls    int
	DeleteServerCalls int
}

func (m *MockClient) CreateServer(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error) {
	m.CreateServerCalls++
	if m.CreateServerFunc == nil {
		m.t.Fatal("CreateServerFunc não implementado no mock")
	}
	return m.CreateServerFunc(ctx, name, imageID, flavorID, userdata, networkIDs, meta)
}

func (m *MockClient) GetServer(ctx context.Context, id string) (ServerInfo, error) {
	m.GetServerCalls++
	if m.GetServerFunc == nil {
		m.t.Fatal("GetServerFunc não implementado no mock")
	}
	return m.GetServerFunc(ctx, id)
}

func (m *MockClient) DeleteServer(ctx context.Context, id string) error {
	m.DeleteServerCalls++
	if m.DeleteServerFunc == nil {
		m.t.Fatal("DeleteServerFunc não implementado no mock")
	}
	return m.DeleteServerFunc(ctx, id)
}