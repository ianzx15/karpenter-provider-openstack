package instance

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/joho/godotenv"

	"github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
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

func TestCreateInstance_Integration(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Erro ao carregar .env: %v", err)
	}

	fmt.Println(os.Getenv("RUN_INTEGRATION_TESTS"))
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Pulando teste de integração. Sete RUN_INTEGRATION_TESTS=1 para executar.")
	}

	ctx := context.Background()

	nodeClass := &v1openstack.OpenStackNodeClass{
		Spec: v1openstack.OpenStackNodeClassSpec{

			Networks: []string{"8e9133dd-0907-42f2-866d-c7ad2af7eb9c"},
			UserData: "#!/bin/bash\necho 'hello from integration test'",
			ImageSelectorTerms: []v1openstack.OpenStackImageSelectorTerm{
				{
					ID: "62dee28f-987d-40f5-a308-051d59991da8",
				},
			},
		},
	}
	nodeClaim := &karpv1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "karpenter-integration-test"},
	}

	instanceType := &cloudprovider.InstanceType{
		Name: "7441c7d9-2648-4a33-907e-4d28c2270da3",
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement("instance-type", "In", "7441c7d9-2648-4a33-907e-4d28c2270da3"),
		),
	}

	// 1. Cria o cliente real
	realComputeClient := createRealComputeClient(t)

	// 2. Cria o Provider e injeta o cliente real
	// (Estou assumindo que seu DefaultProvider tem um campo 'computeClient')
	// Se NewProvider não configurar o cliente, fazemos manualmente:
	testProvider := &DefaultProvider{
		computeClient: realComputeClient, // Injeção do cliente real
		clusterName:   "test-cluster",
		// kubeClient: nil, // Pode ser nil se o 'Create' não usar
	}

	// 3. Registra a função de limpeza
	// Isso garante que a VM seja deletada DEPOIS que o teste rodar
	t.Cleanup(func() {
		t.Log("Iniciando limpeza: procurando por instâncias com nome 'karpenter-integration-test-...'")

		// Lista todas as instâncias para encontrar a que acabamos de criar
		// É melhor do que confiar no 'instance.InstanceID' caso o 'Create' falhe
		listOpts := servers.ListOpts{
			Name: fmt.Sprintf("^%s-", nodeClaim.Name), // Filtra por prefixo
		}
		allPages, err := servers.List(realComputeClient, listOpts).AllPages()
		if err != nil {
			t.Logf("Erro ao listar instâncias para limpeza: %v", err)
			return
		}

		allServers, err := servers.ExtractServers(allPages)
		if err != nil {
			t.Logf("Erro ao extrair instâncias para limpeza: %v", err)
			return
		}

		if len(allServers) == 0 {
			t.Log("Nenhuma instância encontrada para limpar.")
			return
		}

		for _, srv := range allServers {
			t.Logf("Deletando instância: %s (ID: %s)", srv.Name, srv.ID)
			err := servers.Delete(realComputeClient, srv.ID).ExtractErr()
			if err != nil {
				t.Logf("ERRO AO DELETAR instância %s: %v", srv.ID, err)
			} else {
				t.Logf("Instância %s deletada com sucesso.", srv.ID)
			}
		}
	})

	// 4. Executa a função
	t.Log("Executando test.Create...")
	instance, err := testProvider.Create(ctx, nodeClass, nodeClaim, []*cloudprovider.InstanceType{instanceType})

	// 5. Verifica os resultados
	if err != nil {
		t.Fatalf("Falha ao criar instância: %v", err)
	}

	if instance == nil {
		t.Fatal("Instância retornada é nula, mas erro também é nulo")
	}

	// Verifica se os dados retornados fazem sentido
	if instance.InstanceID == "" {
		t.Error("InstanceID está vazio")
	}
	if instance.Type != "general.small" {
		t.Errorf("Tipo incorreto: esperado='general.small', obtido='%s'", instance.Type)
	}
	if instance.ImageID != nodeClass.Spec.ImageSelectorTerms[0].ID {
		t.Errorf("ImageID incorreto: esperado='%s', obtido='%s'", nodeClass.Spec.ImageSelectorTerms[0].ID, instance.ImageID)
	}

	t.Logf("Instância criada com sucesso: ID=%s, Type=%s, ImageID=%s, Status=%s",
		instance.InstanceID, instance.Type, instance.ImageID, instance.Status)

	// O status inicial é geralmente BUILD
	if instance.Status != "BUILD" && instance.Status != "ACTIVE" {
		t.Logf("Aviso: Status inesperado: %s", instance.Status)
	}
}
