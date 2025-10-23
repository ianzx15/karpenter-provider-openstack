package openstack

import (
	"context"
	"log"
	// "os" // REMOVIDO - "os" imported and not used

	"sigs.k8s.io/karpenter/pkg/controllers/interruption"
	"sigs.k8s.io/karpenter/pkg/controllers/nodepool"
	"sigs.k8s.io/karpenter/pkg/controllers/provisioning"
	"sigs.k8s.io/karpenter/pkg/controllers/termination"
	"sigs.k8s.io/karpenter/pkg/webhooks"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// --- IMPORTAÇÕES CORRIGIDAS ---
	"sigs.k8s.io/karpenter/pkg/cloudprovider"                 // ADICIONADO - para cloudprovider.Register
	openstack "github.com/ianzx15/karpenter-provider-openstack/openstack" // ADICIONADO - para o seu provider
)

func main() {
	ctx := context.Background() // Contexto para o provider
	cfg := ctrl.GetConfigOrDie()

	// TODO: Carregar a configuração (ImageID, FlavorID, etc) de um ConfigMap ou env
	// Por enquanto, está hardcoded como exemplo
	osConfig := openstack.Config{
		ImageID:  "seu-image-id",
		FlavorID: "seu-flavor-id",
		NetworkIDs: []string{"sua-network-id"},
		Zone:     "sua-zona",
	}

	// ADICIONADO: Criar o cliente OpenStack
	osClient, err := openstack.NewClient()
	if err != nil {
		log.Fatalf("failed to create openstack client, %v", err)
	}

	// ADICIONADO: Criar o CloudProvider
	cp := openstack.NewCloudProvider(osClient, osConfig)

	opts := manager.Options{
		// ... (suas opções de manager) ...
	}

	// CORRIGIDO: manager.New precisa de (cfg, opts)
	mgr, err := manager.New(cfg, opts)
	if err != nil {
		log.Fatalf("failed to create manager, %v", err)
	}

	// ADICIONADO: Registrar o provider no Karpenter
	if err := cloudprovider.Register(ctx, cp); err != nil {
		log.Fatalf("failed to register cloud provider, %v", err)
	}

	// Registrar os controladores do Karpenter
	if err := provisioning.NewController(mgr.GetClient(), mgr.GetEventRecorderFor("provisioning"), cp).Builder(ctx, mgr).Complete(mgr); err != nil {
		log.Fatalf("failed to create provisioning controller, %v", err)
	}
	if err := termination.NewController(mgr.GetClient(), mgr.GetEventRecorderFor("termination"), cp).Builder(ctx, mgr).Complete(mgr); err != nil {
		log.Fatalf("failed to create termination controller, %v", err)
	}
	if err := interruption.NewController(mgr.GetClient()).Builder(ctx, mgr).Complete(mgr); err != nil {
		log.Fatalf("failed to create interruption controller, %v", err)
	}
	if err := nodepool.NewController(mgr.GetClient()).Builder(ctx, mgr).Complete(mgr); err != nil {
		log.Fatalf("failed to create nodepool controller, %v", err)
	}
	if err := webhooks.New(mgr.GetClient()).Builder(mgr).Complete(mgr); err != nil {
		log.Fatalf("failed to create webhooks, %v", err)
	}

	// Iniciar o manager
	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("failed to start manager, %v", err)
	}