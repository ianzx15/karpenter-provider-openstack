package v1openstack

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

const (
    GroupName = "karpenter.k8s.openstack"
)

var (
    SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1openstack"}

    SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
    AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
    scheme.AddKnownTypes(
        SchemeGroupVersion,
        &OpenStackNodeClass{},
        &OpenStackNodeClassList{},
    )
    metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
    return nil
}