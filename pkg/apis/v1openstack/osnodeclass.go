package v1openstack

import (
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=osnc
// +kubebuilder:subresource:status
type OpenStackNodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackNodeClassSpec   `json:"spec,omitempty"`
	Status OpenStackNodeClassStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen=true
type OpenStackNodeClassSpec struct {
	// Flavor defines the OpenStack flavor to use for the node.
	// Flavor string `json:"flavor"`

	// KeyPair is the OpenStack key pair name to assign to the instance
	// +optional
	KeyPair string `json:"keyPair,omitempty"`

	// Disks defines the disks to attach to the provisioned instance.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Disks []Disk `json:"disks,omitempty"`

	// UserData to be passed to the instance (cloud-init).
	// +optional
	UserData string `json:"userData,omitempty"`

	// ImageRef is the OpenStack Glance image ID to use for the instance.
	// +optional
	ImageRef string `json:"imageRef,omitempty"`

	// ImageSelectorTerms is a list of image selector terms. The terms are ORed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=30
	ImageSelectorTerms []OpenStackImageSelectorTerm `json:"imageSelectorTerms"`

	// Networks specifies the OpenStack networks to attach to the instance.
	// +kubebuilder:validation:MinItems=1
	Networks []string `json:"networks"`

	// SecurityGroups specifies the OpenStack security groups to assign to the instance.
	// +optional
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// FloatingIP indicates whether to assign a floating IP to the instance.
	// +optional
	FloatingIP bool `json:"floatingIP,omitempty"`

	// KubeletConfiguration defines args to be used when configuring kubelet on provisioned nodes.
	// +optional
	KubeletConfiguration *KubeletConfiguration `json:"kubeletConfiguration,omitempty"`

	// Labels to be applied on the OpenStack VM instance.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Metadata contains key/value pairs to set as instance metadata.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// +k8s:deepcopy-gen=true
type OpenStackImageSelectorTerm struct {
	// Alias specifies the image name or family in OpenStack Glance.
	// +kubebuilder:validation:MaxLength=60
	// +optional
	Alias string `json:"alias,omitempty"`

	// ID specifies the exact Glance image ID to use.
	// +kubebuilder:validation:MaxLength=160
	// +optional
	ID string `json:"id,omitempty"`
}

// +k8s:deepcopy-gen=true
type KubeletConfiguration struct {
	// ClusterDNS is a list of IP addresses for the cluster DNS server.
	// +optional
	ClusterDNS []string `json:"clusterDNS,omitempty"`

	// MaxPods is an override for the maximum number of pods that can run on a worker node.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxPods *int32 `json:"maxPods,omitempty"`

	// PodsPerCore is an override for the number of pods per CPU core.
	// +kubebuilder:validation:Minimum=0
	// +optional
	PodsPerCore *int32 `json:"podsPerCore,omitempty"`

	// SystemReserved contains resources reserved for OS system daemons and kernel memory.
	// +optional
	SystemReserved map[string]string `json:"systemReserved,omitempty"`

	// KubeReserved contains resources reserved for Kubernetes system components.
	// +optional
	KubeReserved map[string]string `json:"kubeReserved,omitempty"`

	// EvictionHard defines hard eviction thresholds.
	// +optional
	EvictionHard map[string]string `json:"evictionHard,omitempty"`

	// EvictionSoft defines soft eviction thresholds.
	// +optional
	EvictionSoft map[string]string `json:"evictionSoft,omitempty"`

	// EvictionSoftGracePeriod defines grace periods for soft eviction thresholds.
	// +optional
	EvictionSoftGracePeriod map[string]metav1.Duration `json:"evictionSoftGracePeriod,omitempty"`

	// EvictionMaxPodGracePeriod is the maximum allowed grace period for terminating pods.
	// +optional
	EvictionMaxPodGracePeriod *int32 `json:"evictionMaxPodGracePeriod,omitempty"`

	// ImageGCHighThresholdPercent is the disk usage percent after which image GC is always run.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	ImageGCHighThresholdPercent *int32 `json:"imageGCHighThresholdPercent,omitempty"`

	// ImageGCLowThresholdPercent is the disk usage percent before which image GC is never run.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	ImageGCLowThresholdPercent *int32 `json:"imageGCLowThresholdPercent,omitempty"`

	// CPUCFSQuota enables CPU CFS quota enforcement for containers that specify CPU limits.
	// +optional
	CPUCFSQuota *bool `json:"cpuCFSQuota,omitempty"`
}

// +k8s:deepcopy-gen=true
type Disk struct {
	// SizeGiB is the size of the disk in GiB.
	// +kubebuilder:validation:Minimum=10
	SizeGiB int32 `json:"sizeGiB"`

	// VolumeType is the Cinder volume type (e.g., standard, ssd, high-speed).
	// +optional
	VolumeType string `json:"volumeType,omitempty"`

	// Boot indicates that this is the boot disk.
	// +optional
	Boot bool `json:"boot,omitempty"`
}

// +k8s:deepcopy-gen=true
type OpenStackNodeClassStatus struct {
	Conditions []status.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
type OpenStackNodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackNodeClass `json:"items"`
}

// StatusConditions retorna um ConditionSet vinculado a este objeto.
// "Ready" é a condição padrão que define se o objeto está saudável.
func (in *OpenStackNodeClass) StatusConditions() status.ConditionSet {
    // 1. NewReadyConditions("Ready") define a regra.
    // 2. .For(in) vincula a regra a ESTA instância do objeto.
    return status.NewReadyConditions("Ready").For(in)
}

// MANTENHA ESTES DOIS ABAIXO (Obrigatórios para o .For() funcionar)
func (in *OpenStackNodeClass) GetConditions() []status.Condition {
    return in.Status.Conditions
}

func (in *OpenStackNodeClass) SetConditions(conditions []status.Condition) {
    in.Status.Conditions = conditions
}