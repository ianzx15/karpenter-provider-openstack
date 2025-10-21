// pkg/utils/userdata.go
package utils

import (
	"encoding/base64"
	"fmt"
)

// GenerateUserData cria um script cloud-init básico para o nó.
// ATENÇÃO: Este é um exemplo mínimo.
func GenerateUserData(clusterName, clusterEndpoint string) string {
	// Script de exemplo. Você precisará de um script real
	// para juntar o nó ao cluster (ex: usando kubeadm).
	script := `#!/bin/bash
echo "ClusterName: %s"
echo "ClusterEndpoint: %s"
echo "Userdata executado com sucesso!"
# Aqui entraria a lógica de 'kubeadm join' ou similar
`

	formattedScript := fmt.Sprintf(script, clusterName, clusterEndpoint)
	return base64.StdEncoding.EncodeToString([]byte(formattedScript))
}