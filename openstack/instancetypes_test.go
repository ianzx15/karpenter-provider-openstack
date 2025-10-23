// pkg/openstack/instancetypes_test.go
package openstack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildInstanceTypes(t *testing.T) {
	cfg := Config{
		Zone: "nova-test-zone",
	}

	its, err := BuildInstanceTypes(cfg)
	require.NoError(t, err)
	require.Len(t, its, 1, "Deveria retornar 1 tipo de instância MVP")

	it := its[0]
	assert.Equal(t, "openstack-mvp", it.Name)
	
	// Verifica Requisitos (Zona)
	expectedReqs := map[string][]string{
		"topology.kubernetes.io/zone": {"nova-test-zone"},
	}
	assert.Equal(t, expectedReqs, it.Requirements)

	// Verifica Recursos (CPU/Memória)
	expectedCPU := resource.NewQuantity(2, resource.DecimalSI)
	expectedMem := resource.NewQuantity(4*1024*1024*1024, resource.BinarySI) // 4Gi

	assert.True(t, expectedCPU.Equal(it.Resources[corev1.ResourceCPU]), "CPU não bate")
	assert.True(t, expectedMem.Equal(it.Resources[corev1.ResourceMemory]), "Memória não bate")
	
	// Verifica Offerings
	require.Len(t, it.Offerings, 1)
	assert.Equal(t, "nova-test-zone", it.Offerings[0].Zone)
	assert.Equal(t, "on-demand", it.Offerings[0].CapacityType)
}