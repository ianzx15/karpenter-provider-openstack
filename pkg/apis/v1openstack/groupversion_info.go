package v1openstack

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion = schema.GroupVersion{
		Group:   "openstack.karpenter.sh",
		Version: "v1alpha1",
	}

	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		GroupVersion,
		&OpenStackNodeClass{},
		&OpenStackNodeClassList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
