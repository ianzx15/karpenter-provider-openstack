id dos flavors:
general.small 7441c7d9-2648-4a33-907e-4d28c2270da3
general.medium 69495bdc-cc5a-4596-9b0a-e2c30956df46
general.large be9875d8-f22b-426e-91e1-79f04c705c09
general.xlarge dfc86c3d-c4c1-4c03-9548-d6dc0b3b42f6

Iniciando ambiente de testes:
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


  