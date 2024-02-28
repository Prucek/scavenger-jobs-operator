package node

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	THRESHOLD = 0.7
)

func getNodePodCapacity(node *corev1.Node) int {
	podCapacity := 0
	for resourceName, allocatable := range node.Status.Allocatable {
		if resourceName == corev1.ResourcePods {
			podCapacity = int(allocatable.Value())
		}
	}
	return podCapacity
}

func IsClusterBusy(ctx context.Context, client client.Client, addend int, log logr.Logger) bool {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods); err != nil {
		log.Error(err, "unable to list pods")
	}
	nodeName := types.NamespacedName{
		Name:      pods.Items[0].Spec.NodeName,
		Namespace: pods.Items[0].Namespace,
	}
	node := corev1.Node{}
	if err := client.Get(ctx, nodeName, &node); err != nil {
		log.Error(err, "unable to get node")
		return false
	}
	maxPodCount := getNodePodCapacity(&node)
	return float64(len(pods.Items)+addend) > float64(maxPodCount)*THRESHOLD
}
