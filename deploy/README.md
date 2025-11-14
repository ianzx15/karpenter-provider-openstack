# 1. Crie o namespace
kubectl create namespace karpenter

# 2. Crie uma secret com suas credenciais do OpenStack
# O seu provider (código Go) deve ler a variável de ambiente OS_CLOUDS ou buscar este arquivo.
cat <<EOF > clouds.yaml
clouds:
  openstack:
    auth:
      auth_url: "https://seu-auth-url:5000/v3"
      username: "seu-usuario"
      password: "sua-senha"
      project_id: "seu-project-id"
      user_domain_name: "Default"
    region_name: "RegionOne"
    interface: "public"
    identity_api_version: 3
EOF

kubectl create secret generic openstack-cloud-config \
  --from-file=clouds.yaml=clouds.yaml \
  --namespace karpenter

# Crie o secret no cluster

kubectl create secret generic openstack-cloud-config \
  --namespace karpenter \
  --from-file=clouds.yaml=clouds.yaml

# Defina a versão do seu build local
export KARPENTER_VERSION="0.0.1-dev"
export IMG_REPOSITORY="seu-docker-registry/karpenter-openstack"


# Instalar o controller-gen:
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

# Isso varre seu código em busca de comentários // +kubebuilder e gera o YAML
controller-gen crd paths="./..." output:crd:artifacts:config=config/crd/bases

# Aplique o arquivo gerado no cluster
kubectl apply -f config/crd/bases/


# Instala NodePool, NodeClaim e outros recursos do Core
kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodeclaims.yaml

kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodepools.yaml




nodeclass:

apiVersion: karpenter.k8s.openstack/v1alpha1 # O group que você definiu no kubebuilder
kind: OpenStackNodeClass
metadata:
  name: default
spec:
  # Referência à rede onde os nós serão criados
  networkID: "uuid-da-sua-rede-privada" 
  # Security Groups que os nós devem ter
  securityGroups:
    - "k8s-worker-sec-group"
    - "default"
  # Tags de metadados para o OpenStack
  tags:
    karpenter.sh/discovery: "seu-cluster-name"
  # User data para bootstrap (join script do kubeadm/k3s/etc)
  userData: |
    #!/bin/bash
    echo "Entrando no cluster..."
    /usr/bin/kubeadm join ...


nodepool:

apiVersion: karpenter.sh/v1
kind: NodePool
metadata:
  name: default
spec:
  template:
    spec:
      requirements:
        - key: kubernetes.io/arch
          operator: In
          values: ["amd64"]
        - key: kubernetes.io/os
          operator: In
          values: ["linux"]
        # Se o seu provider suportar filtragem de flavor por vCPU/RAM:
        - key: karpenter.k8s.aws/instance-category 
          operator: DoesNotExist # Remova chaves específicas da AWS se seu provider não as implementou ainda
      nodeClassRef:
        group: karpenter.k8s.openstack # Seu grupo
        kind: OpenStackNodeClass       # Seu Kind
        name: default
  limits:
    cpu: 1000
  disruption:
    consolidationPolicy: WhenEmptyOrUnderutilized
    consolidateAfter: 1m



kubectl delete crd nodepools.karpenter.sh nodeclaims.karpenter.sh openstacknodeclasses.karpenter.k8s.openstack --ignore-not-found=true

# Instala os CRDs oficiais do projeto upstream
kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodeclaims.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodepools.yaml

go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

~/go/bin/controller-gen crd paths="./..." output:stdout | kubectl apply -f -

source .env

export KUBERNETES_MIN_VERSION="1.19.0"
export KARPENTER_SERVICE="karpenter"
export LOG_LEVEL="debug"
export OS_CLOUD="openstack" # Nome da nuvem no seu clouds.yaml (se usar arquivo)
# Desabilita validação de webhook por enquanto para rodar local fácil
export DISABLE_WEBHOOK="true"


# gerar o zz_generated.deepcopy.go:
controller-gen object paths="./..."


# Comando para tornar true o nodepool
kubectl patch openstacknodeclasses.karpenter.k8s.openstack default --type=merge --subresource=status -p '{"status":{"conditions":[{"type":"Ready","status":"True","lastTransitionTime":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","reason":"Reconciled","message":"Manually set to Ready"}]}}'


# deletando tudo
kubectl delete nodeclaim --all
kubectl delete deployment --all
kubectl delete nodepool --all

# criando recurso
kubectl apply -f dev-resources.yaml

kubectl apply -f small-pod.yaml