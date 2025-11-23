
## 1. PREPARAÇÃO DO SISTEMA, CONTAINERD E BINÁRIOS DO K8S (v1.30)


### 1. Desabilita o swap
```
sudo swapoff -a; sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
```
### 2. Configura módulos de kernel e network
```
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay; sudo modprobe br_netfilter
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sudo sysctl --system
```
### 3. Instala Containerd
```
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update 
sudo apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release containerd.io
sudo mkdir -p /etc/containerd; sudo containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml
sudo systemctl restart containerd; sudo systemctl enable containerd
```
### 4. Instalação dos binários do K8s (v1.30)
```
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update 
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```
## 5. INICIALIZAÇÃO, ACESSO E REDE (FLANNEL)

### 5. Inicializa o nó de controle (ajuste o IP se necessário)
```
sudo kubeadm init --apiserver-advertise-address=<host_ip> --pod-network-cidr=10.244.0.0/16 --control-plane-endpoint=<host_ip>
```

### 6. Configura o kubectl para o seu usuário
```
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```


### 7. Instala o Flannel CNI (ignora erros de certificado/validação após init)
```
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml --validate=false --insecure-skip-tls-verify
```

## 8. VERIFICAÇÃO E JOIN COMMAND (OPCIONAL)

### 8. Verifica se o nó está "Ready"
```
kubectl --insecure-skip-tls-verify get nodes
```
### 9. Gera o comando para adicionar nós de trabalho
```
kubeadm token create --print-join-command
```
### 10. Criando namespace e secrets
```
kubectl create namespace karpenter
kubectl create secret generic openstack-cloud-config     --from-file=cloud.yaml=./cloud.yaml     -n karpenter
secret/openstack-cloud-config created
```

## Conectando os worker nodes

### Execute o seguinte comando no worker node para adicioná-lo ao cluster
sudo kubeadm join 10.5.8.192:6443 --token iqze4k.9dxpmgm306zbfdc9 --discovery-token-ca-cert-hash sha256:fd05c1c4d86e676e53b093a991ec1bcd5e7e94e79ff5dd30b7e294fe2099caa9 
