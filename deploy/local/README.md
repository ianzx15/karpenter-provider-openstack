# Guia de Desenvolvimento: Karpenter Provider para OpenStack

Este guia mostra os passos para configurar um ambiente de desenvolvimento e rodar o controller do Karpenter (provider OpenStack) localmente.

## 1. Pré-requisitos

Certifique-se de que possui as seguintes ferramentas instaladas e configuradas:

* **Go** (v1.21+)
* **kubectl** (configurado para seu cluster de dev)
* **controller-gen** (para gerar CRDs e código `deepcopy`):
    ```bash
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    ```

## 2. Configuração de Credenciais

O controller precisa de credenciais para se comunicar com a API do OpenStack.

1.  **Crie o `clouds.yaml`** (substitua pelos seus valores reais):
    ```yaml
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
    ```

2.  **Crie o Cluster, Namespace e o Secret** no cluster:
    ```bash
    kind create cluster --name karpenter-dev
    
    kubectl create namespace karpenter
    
    kubectl create secret generic openstack-cloud-config \
      --from-file=clouds.yaml=clouds.yaml \
      --namespace karpenter
    ```

## 3. Preparando o Cluster (CRDs)

Precisamos instalar as "definições de recursos" (CRDs) que o Karpenter usa.

1.  **(Opcional) Limpeza:** Remova definições antigas de testes anteriores.
    ```bash
    kubectl delete crd nodepools.karpenter.sh nodeclaims.karpenter.sh --ignore-not-found=true
    kubectl delete crd openstacknodeclasses.karpenter.k8s.openstack --ignore-not-found=true 
    ```

2.  **Instale os CRDs do Karpenter Core (v1):**
    O provider `v1` exige os CRDs `v1` do core.
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodepools.yaml

    kubectl apply -f https://raw.githubusercontent.com/aws/karpenter-provider-aws/v1.0.0/pkg/apis/crds/karpenter.sh_nodeclaims.yaml
    ```

3.  **No root do sistema gere e Instale o seu `OpenStackNodeClass` CRD:**
    Este comando lê seu código Go (`pkg/apis/...`) e instala a definição customizada no cluster.
    ```bash
    # (Opcional) Gere o 'zz_generated.deepcopy.go' se você alterou as structs
    controller-gen object paths="./..."

    # Gere e aplique o CRD YAML (ajuste a versão se necessário)
    controller-gen crd paths="./..." output:stdout | kubectl apply -f -
    ```

## 4. Rodando o Controller Localmente

Em vez de usar Docker, rode o controller direto do seu terminal para ver os logs em tempo real.

1.  **Crie o arquivo .env com o seguinte conteúdo:**
    ```bash
    export OS_AUTH_URL="https://your-url"
    export OS_USERNAME="your-user"
    export OS_PASSWORD="your-password"
    export OS_PROJECT_NAME="your-project-name"
    export OS_DOMAIN_NAME="your-domain"
    export OS_REGION_NAME="your-zone"

    export CLUSTER_NAME="karpenter-openstack-test"

    export LEADER_ELECTION_NAMESPACE=kube-system

    export KUBERNETES_MIN_VERSION="1.19.0"
    export KARPENTER_SERVICE="karpenter"
    export LOG_LEVEL="debug"
    export DISABLE_WEBHOOK="true"

    export RUN_INTEGRATION_TESTS=1 
    ```
2. **Aplique ao terminal:**
    ```
    source .env
    ```

3.  **Inicie o Controller:**
    (Mantenha este terminal aberto para ver os logs)
    ```bash
    go run cmd/controller/main.go
    ```

## 5. Testando o Provisionamento

Em um **novo terminal**, vamos criar os recursos que disparam o provisionamento.

1.  **Crie o `dev-resources.yaml`:**
    * Certifique-se de que `apiVersion` bate com o que foi gerado no passo 3 (provavelmente `.../v1openstack`).
    * **Use os UUIDs reais** do seu OpenStack para a imagem e rede.

    ```yaml
    apiVersion: karpenter.k8s.openstack/v1openstack
    kind: OpenStackNodeClass
    metadata:
      name: default
    spec:
      # Use o ID da imagem, não o alias
      imageSelectorTerms:
        - id: "UUID-DA-SUA-IMAGEM-GLANCE" 
      # Use o ID da rede
      networks:
        - "UUID-DA-SUA-REDE-NEUTRON"
      securityGroups:
        - "default"
      metadata:
        karpenter.sh/discovery: "karpenter-cluster"
      # userData: |
      #   #!/bin/bash
      #   /usr/bin/kubeadm join ...
    
    ---
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
          nodeClassRef:
            group: karpenter.k8s.openstack # O mesmo grupo do NodeClass
            kind: OpenStackNodeClass
            name: default
      limits:
        cpu: 10
      disruption:
        consolidationPolicy: WhenEmptyOrUnderutilized
        consolidateAfter: 1m
    ```

2.  **Crie o `test-pod.yaml` (para teste):**
    * Peça recursos que caibam no seu flavor `general.medium` (que tem 2 vCPU - 0.5 overhead = 1.5 vCPU livre).

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: test-inflate
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: inflate
      template:
        metadata:
          labels:
            app: inflate
        spec:
          terminationGracePeriodSeconds: 0
          containers:
            - name: inflate
              image: public.ecr.aws/eks-distro/kubernetes/pause:3.7
              resources:
                requests:
                  cpu: "1.5" # Cabe no 'medium'
                  memory: "1Gi"
    ```

3.  **Aplique os Recursos:**
    ```bash
    kubectl apply -f dev-resources.yaml
    kubectl apply -f small-pod.yaml
    ```

4.  **Observe o terminal do `go run`:** Você deve ver os logs `Instance successfully created`!

## 6. Comandos Úteis (Debug)

### Forçar o NodePool a ficar "Ready"
Se o `kubectl get nodepool` mostrar `READY=False` ou `Unknown` (e o controller não tiver erros), o status do `NodeClass` pode não ter atualizado. Force-o:
```bash
# Certifique-se de usar o nome correto do CRD (com .karpenter.k8s.openstack)
kubectl patch openstacknodeclasses.karpenter.k8s.openstack default --type=merge --subresource=status -p '{"status":{"conditions":[{"type":"Ready","status":"True","lastTransitionTime":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","reason":"Reconciled","message":"Manually set"}]}}'
```


### Limpeza do ambiente
```
kubectl delete nodeclaim --all
kubectl delete deployment --all
kubectl delete nodepool --all
```


### Deletando nodeclaims zumbis
**Execute no terminal:**
```
for nc in $(kubectl get nodeclaims -o name); do
  echo "Forçando a exclusão de ${nc}..."
  kubectl patch ${nc} -p '{"metadata":{"finalizers":null}}' --type=merge
done

```