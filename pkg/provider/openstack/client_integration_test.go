//go:build integration
// +build integration

// pkg/openstack/client_integration_test.go
package openstack

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestConfig carrega a configuração real do ambiente para o teste
func getTestConfig(t *testing.T) (imageID, flavorID, networkID string) {
	imageID = os.Getenv("TEST_OS_IMAGE_ID")
	flavorID = os.Getenv("TEST_OS_FLAVOR_ID")
	networkID = os.Getenv("TEST_OS_NETWORK_ID") // Seu código espera uma string separada por vírgula

	if imageID == "" || flavorID == "" || networkID == "" {
		t.Skip("Pulando teste de integração: TEST_OS_IMAGE_ID, TEST_OS_FLAVOR_ID ou TEST_OS_NETWORK_ID não definidos")
	}
	return
}

// TestClient_Lifecycle testa o ciclo de vida completo: Create -> Get -> Delete
func TestClient_Lifecycle(t *testing.T) {
	imageID, flavorID, networkID := getTestConfig(t)
	
	ctx := context.Background()
	client, err := NewClient() // Usa credenciais reais do .env
	require.NoError(t, err, "Falha ao criar o cliente OpenStack")

	serverName := "karpenter-integ-test"
	serverID := "" // Para garantir a limpeza

	// t.Cleanup() garante que o DeleteServer será chamado mesmo se o teste falhar
	t.Cleanup(func() {
		if serverID != "" {
			t.Logf("Limpando servidor %s", serverID)
			err := client.DeleteServer(ctx, serverID)
			if err != nil {
				// Loga o erro mas não falha o teste,
				// pois o teste principal pode já ter falhado.
				t.Logf("AVISO: Falha ao limpar servidor %s: %v", serverID, err)
			}
		}
	})

	// 1. Criar Servidor
	t.Log("Criando servidor...")
	serverID, err = client.CreateServer(
		ctx,
		serverName,
		imageID,
		flavorID,
		"#!/bin/bash\necho 'hello world'", // userdata
		[]string{networkID},
		map[string]string{"karpenter-test": "true"},
	)
	require.NoError(t, err, "Falha ao chamar CreateServer")
	require.NotEmpty(t, serverID, "CreateServer retornou um ID vazio")
	t.Logf("Servidor criado com ID: %s", serverID)

	// 2. Poll GetServer até ficar ATIVO
	var srvInfo ServerInfo
	timeout := time.After(5 * time.Minute) // Timeout de 5 min
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	t.Log("Aguardando servidor ficar ACTIVE...")
L:
	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout: Servidor %s não ficou ACTIVE a tempo", serverID)
		case <-ticker.C:
			srvInfo, err = client.GetServer(ctx, serverID)
			require.NoError(t, err, "Falha ao chamar GetServer")
			t.Logf("Status atual: %s", srvInfo.Status)
			if srvInfo.Status == "ACTIVE" {
				break L
			}
			if srvInfo.Status == "ERROR" {
				t.Fatalf("Servidor %s entrou em estado de ERROR", serverID)
			}
		}
	}
	
	// 3. Verificar Informações do GetServer
	assert.Equal(t, serverID, srvInfo.ID)
	assert.Equal(t, serverName, srvInfo.Name)
	assert.Equal(t, "ACTIVE", srvInfo.Status)
	assert.NotEmpty(t, srvInfo.IPs, "Servidor ATIVO deveria ter IPs")
	assert.Greater(t, srvInfo.CPU, int64(0), "CPU do Flavor não foi populada")
	assert.Greater(t, srvInfo.Memory, int64(0), "Memória do Flavor não foi populada")

	// 4. Deletar Servidor (será chamado pelo t.Cleanup())
	t.Log("Teste de Get/Create concluído, t.Cleanup() cuidará da deleção.")
}