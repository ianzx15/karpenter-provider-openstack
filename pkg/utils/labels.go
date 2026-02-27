package utils

import (
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/karpenter/pkg/cloudprovider"
)

func GetAllSingleValuedRequirementLabels(it *cloudprovider.InstanceType) map[string]string {
	labels := map[string]string{}
	for _, req := range it.Requirements.Values() {

		if req.Operator() == corev1.NodeSelectorOpIn && len(req.Values()) == 1 {

			labels[req.Key] = req.Values()[0]
		}
	}

	return labels
}
