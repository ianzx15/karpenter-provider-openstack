/*
Copyright 2025 Seu Nome/Sua Empresa.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package instance gerencia a lógica de criação e exclusão de instâncias no OpenStack.
package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"


)

// TODO: Defina sua própria estrutura OpenStackNodeClass em seu pacote de API.
// Este é apenas um exemplo de como poderia ser.
/*
type OpenStackNodeClass struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   OpenStackNodeClassSpec   `json:"spec,omitempty"`
    Status OpenStackNodeClassStatus `json:"status,omitempty"`
}

type OpenStackNodeClassSpec struct {
    // ID da imagem a ser usada para o boot da instância (ex: Glance Image ID)
    ImageID string `json:"imageID"`
    // ID da rede a ser conectada à instância
    NetworkID string `json:"networkID"`
    // Zona de Disponibilidade onde a instância será criada
    AvailabilityZone string `json:"availabilityZone,omitempty"`
    // Tags a serem aplicadas na instância
    Tags map[string]string `json:"tags,omitempty"`
    // Script de inicialização da instância
    UserData string `json:"userData,omitempty"`
}
*/


type Provider interface {
	Create(context.Context, *v1alpha1.OpenStackNodeClass, *karpv1.NodeClaim, *cloudprovider.InstanceType) (*Instance, error)
}

// OpenStackProvider implementa a interface Provider.
// type OpenStackProvider struct {
// 	// TODO: openstackClient será o seu cliente para o SDK do OpenStack.
// 	// openstackClient *gophercloud.ServiceClient
// 	clusterName string
// }

// NewProvider cria um novo OpenStackProvider.
func NewProvider(clusterName string, client *gophercloud.ServiceClient) *OpenStackProvider {
	return &OpenStackProvider{
		clusterName: clusterName,
		openstackClient: client,
	}
}

// Create é a função principal do MVP. Ela cria uma única instância no OpenStack.
func (p *OpenStackProvider) Create(ctx context.Context, nodeClass *v1alpha1.OpenStackNodeClass, nodeClaim *karpv1.NodeClaim, instanceType *cloudprovider.InstanceType) (*Instance, error) {
	logger := log.FromContext(ctx)

	// MVP Simplification: Usamos diretamente as especificações do nodeClass.
	// Não há seleção complexa de zonas ou templates.
	instanceName := fmt.Sprintf("karpenter-%s", nodeClaim.Name)
	flavorName := instanceType.Name // O nome do InstanceType será o nome do Flavor no OpenStack.
	
	logger.Info("Iniciando criação de instância no OpenStack",
		"name", instanceName,
		"flavor", flavorName,
		"image", nodeClass.Spec.ImageID,
		"network", nodeClass.Spec.NetworkID,
		"az", nodeClass.Spec.AvailabilityZone)

	
	createOpts := servers.CreateOpts{
		Name:             instanceName,
		FlavorName:       flavorName,
		ImageRef:         nodeClass.Spec.ImageID,
		AvailabilityZone: nodeClass.Spec.AvailabilityZone,
		Networks:         []servers.Network{{UUID: nodeClass.Spec.NetworkID}},
		UserData:         []byte(nodeClass.Spec.UserData),
		Metadata:         lo.Assign(nodeClass.Spec.Tags, map[string]string{
			"karpenter.sh/nodepool":      nodeClaim.Labels[karpv1.NodePoolLabelKey],
			"karpenter.sh/cluster-name":  p.clusterName,
		}),
	}
	
	// TODO: Chamar a API do OpenStack para criar o servidor (instância).
	createdServer, err := servers.Create(p.openstackClient, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("falha ao chamar a API de criação de instância do OpenStack: %w", err)
	}
	
	logger.Info("Instância enviada para criação", "id", createdServer.ID)
	
	// TODO: Implementar uma lógica de espera para a instância ficar "ACTIVE".
	// O ideal é usar `waitFor.Server`, do próprio gophercloud, ou um loop simples.
	// err = waitForInstanceActive(ctx, p.openstackClient, createdServer.ID, 5*time.Minute)
	// if err != nil {
	// 	return nil, fmt.Errorf("instância '%s' não atingiu o estado ACTIVE: %w", createdServer.ID, err)
	// }

	*/
	
	// Para o MVP, vamos simular uma criação bem-sucedida e retornar um objeto `Instance`.
	// Substitua esta parte pela chamada real da API.
	mockCreatedServerID := "mock-instance-id-" + string(nodeClaim.UID[:8])
	logger.Info("Criação de instância simulada com sucesso", "id", mockCreatedServerID)

	// Retorna a estrutura `Instance` que o Karpenter espera.
	return &Instance{
		InstanceID:   mockCreatedServerID,
		ProviderID:   "openstack:///" + mockCreatedServerID, // Formato do provider ID
		Name:         instanceName,
		Type:         flavorName,
		Location:     nodeClass.Spec.AvailabilityZone,
		ImageID:      nodeClass.Spec.ImageID,
		CreationTime: time.Now(),
		Status:       "PROVISIONING", // O status inicial
	}, nil
}
