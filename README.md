

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


  
