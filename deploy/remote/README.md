-----

# 🚀 Guia de Implantação: Cluster Kubernetes (v1.30) com Containerd e Karpenter (Provider OpenStack)

Este guia cobre a preparação do sistema, a instalação do Kubernetes Control Plane (nó mestre), a configuração do CNI (Flannel) e a preparação do ambiente para o Karpenter com OpenStack.

## 1\. ⚙️ Pré-requisitos e Configuração de Arquivos

Estes arquivos contêm informações sensíveis e de configuração. Eles serão usados para criar Secrets no Kubernetes.

### 📄 1.1. Arquivo `clouds.yaml` (Configuração OpenStack)

Este arquivo será usado para criar o Secret principal que o provedor Karpenter OpenStack consumirá.

```yaml
# clouds.yaml
clouds:
  openstack:
    auth:
      auth_url: "https://teste"
      username: "teste"
      password: "teste" # CORRIGIR: Usar sua senha real aqui.
      project_id: "teste"
      user_domain_name: "teste"
      # Não é necessário project_domain_name se user_domain_name for especificado
    region_name: "teste"
    interface: "public" # Alterado para 'public' como padrão para APIs
    identity_api_version: 3
```

### 📄 1.2. Arquivo `os-env-vars.yaml` (Secret de Variáveis de Ambiente)

**Nota:** Este Secret **não é utilizado** na implantação padrão do Karpenter e é **redundante** se você usar o `clouds.yaml`. Recomenda-se eliminá-lo para simplificar, mas mantemos o conteúdo apenas para referência:

```yaml
# os-env-vars.yaml (Recomendado DELETAR)
apiVersion: v1
kind: Secret
metadata:
  name: os-env-vars
  namespace: karpenter
type: Opaque
stringData:
  OS_AUTH_URL: "https://teste"
  OS_USERNAME: "teste"
  OS_PASSWORD: "@senha"
  OS_REGION_NAME: "teste"
  OS_PROJECT_NAME: "teste"
  OS_USER_DOMAIN_NAME: "teste"
  OS_PROJECT_DOMAIN_NAME: "teste"
  OS_PROJECT_ID: "teste"
```

-----

## 2\. 🛠️ Preparação do Sistema Operacional (No(s) Nó(s) Control Plane e Worker)

Execute os comandos abaixo **em todas as máquinas** que farão parte do cluster (Control Plane e Workers).

### 2.1. Desabilitar o Swap

O Kubernetes (kubelet) exige que o swap esteja desativado.

```bash
sudo swapoff -a; sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
```

### 2.2. Configurar Módulos do Kernel e Network

Garantir que os módulos `overlay` e `br_netfilter` estejam carregados e que as configurações de rede do kernel sejam adequadas para o CNI.

```bash
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay
sudo modprobe br_netfilter
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sudo sysctl --system
```

### 2.3. Instalar Containerd

O Containerd será usado como o *Container Runtime Interface (CRI)*.

```bash
# Adicionar chaves e repositório do Docker
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Instalar Containerd e dependências
sudo apt-get update 
sudo apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release containerd.io

# Configurar Containerd
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml
# Habilitar SystemdCgroup
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml

# Reiniciar e habilitar o serviço
sudo systemctl restart containerd
sudo systemctl enable containerd
```

### 2.4. Instalar Binários do K8s (v1.30)

Instalar o `kubelet`, `kubeadm` e `kubectl`.

```bash
# Adicionar chaves e repositório do K8s v1.30
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list

# Instalar e travar a versão
sudo apt-get update 
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

-----

## 3\. ☸️ Inicialização do Control Plane (Apenas no Nó Mestre)

Execute os comandos abaixo **apenas no Nó Mestre** (Control Plane).

### 3.1. Inicializar o Control Plane

**Importante:** Ajuste o `<host_ip>` para o endereço IP do seu Nó Mestre.

```bash
# Substitua <host_ip> pelo IP do nó mestre
sudo kubeadm init \
    --apiserver-advertise-address=<host_ip> \
    --pod-network-cidr=10.244.0.0/16 \
    --control-plane-endpoint=<host_ip>
```

### 3.2. Configurar o `kubectl`

Permite que seu usuário interaja com o cluster (Copie e execute o bloco completo).

```bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

### 3.3. Instalar o Flannel CNI

O Flannel é o *Container Network Interface (CNI)* que permite a comunicação entre os Pods.

```bash
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

**Nota:** Removemos as flags `--validate=false --insecure-skip-tls-verify`, pois elas são geralmente desnecessárias para o Flannel e reduzem a segurança.

-----

## 4\. 🔬 Verificação e Adição de Workers

Execute os comandos de verificação no Nó Mestre.

### 4.1. Verificar o Status do Nó

Aguarde até que o nó esteja com o status `Ready` (pode levar alguns minutos).

```bash
kubectl get nodes
```

### 4.2. Adicionar Nós de Trabalho (Workers)

Use este comando para gerar o *join command* e executá-lo em todos os seus **Workers**.

```bash
sudo kubeadm token create --print-join-command
```

**Exemplo de Join Command (Executar no Worker):**

```bash
sudo kubeadm join 10.5.8.192:6443 --token iqze4k.9dxpmgm306zbfdc9 --discovery-token-ca-cert-hash sha256:fd05c1c4d86e676e53b093a991ec1bcd5e7e94e79ff5dd30b7e294fe2099caa9
```

-----

## 5\. ☁️ Configuração do Karpenter (Provedor OpenStack)

Agora configuraremos o ambiente para instalar o Karpenter e o provedor OpenStack.

### 5.1. Criar Namespace e Secrets

Cria o namespace `karpenter` e o secret `openstack-cloud-config` a partir do arquivo `clouds.yaml`.

```bash
kubectl create namespace karpenter

# Cria o secret usando o arquivo clouds.yaml da Seção 1.1
kubectl create secret generic openstack-cloud-config \
    --from-file=clouds.yaml=./clouds.yaml \
    --namespace karpenter
```

### 5.2. Instalar Karpenter (Core) e OpenStack Provider

Você listou dois comandos `helm install/upgrade`. O correto é instalar o Karpenter Core e, em seguida, instalar o provedor OpenStack separadamente.

#### A. Instalar o Karpenter (Core)

**CORRIGIDO:** O comando `helm install` estava incompleto e o `helm upgrade` usava repositórios da AWS (ECR) que não são apropriados para a instalação base do Karpenter (que agora é um Chart OCI). Use o repositório Helm padrão e defina os parâmetros corretos.

```bash
# Adicionar repositório do Karpenter
helm repo add karpenter https://charts.karpenter.sh/
helm repo update

# Instalar o Karpenter Core
helm install karpenter karpenter/karpenter \
  --namespace karpenter \
  --create-namespace \
  --set settings.clusterName=kubernetes \
  --set settings.clusterEndpoint="$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" 
```

#### B. Instalar o Karpenter OpenStack Provider

Você precisará do chart Helm oficial do provedor OpenStack. **Você precisará obter o nome e a versão corretos do chart OCI do provedor OpenStack** (O trecho `helm upgrade --install karpenter oci://public.ecr.aws/karpenter/karpenter` não é o provedor OpenStack).

O provedor é normalmente implantado como um Deployment simples (como no seu YAML inicial do outro prompt), mas se for via Helm, você deve usar o nome do chart do provedor.

**Recomendação:** Se você já tem os YAMLs do seu primeiro prompt, **não use Helm**, use apenas o `kubectl apply -f`:

```bash
# SE VOCÊ JÁ TEM O YAML DE DEPLOY DO PROVEDOR:
# Crie o deployment do provedor OpenStack usando seu arquivo YAML original
kubectl apply -f openstack-provider-deploy.yaml
```


# Anotações:
Problema:
Os nós não estão sendo adicionados automaticamente.
Adicionar um nó manualmente (ian-manel-tcc-3) permite ao karpenter alocar o pod nesse nó de forma automatiazada.