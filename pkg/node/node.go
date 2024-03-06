package node

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var previousCapacities = []float64{}

const (
	RESERVE = 0.1
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

func getNodesPodCount(ctx context.Context, client client.Client, log logr.Logger) (int, int, error) {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods); err != nil {
		log.Error(err, "unable to list pods")
		return 0, 0, err
	}

	nodes := &corev1.NodeList{}
	totalPodCapacity := 0
	if err := client.List(ctx, nodes); err != nil {
		log.Error(err, "unable to list nodes")
		return 0, 0, err
	}
	for _, node := range nodes.Items {
		totalPodCapacity += getNodePodCapacity(&node)
	}

	currentPodCount := len(pods.Items)
	return totalPodCapacity, currentPodCount, nil
}

func updatePreviousUsage(previousUsage *[]float64, ctx context.Context, client client.Client, log logr.Logger) {
	totalPodCapacity, currentpodCount, err := getNodesPodCount(ctx, client, log)
	if err != nil {
		log.Error(err, "unable to get nodes")
	}
	currentUsage := float64(currentpodCount) / float64(totalPodCapacity)
	if len(*previousUsage) < 9 {
		*previousUsage = append(*previousUsage, currentUsage)
	} else {
		*previousUsage = append((*previousUsage)[1:], currentUsage)
	}
}

func getThresholds(previousUsage []float64, reserve float64) (float64, float64) {
	stdev, err := stats.StandardDeviation(previousUsage)

	if err != nil {
		stdev = 0
	}
	return 1 - (reserve + stdev), 1 - (reserve/3 + stdev)
}

func IsClusterBusy(ctx context.Context, client client.Client, log logr.Logger) (bool, bool) {
	updatePreviousUsage(&previousCapacities, ctx, client, log)
	bottomT, topT := getThresholds(previousCapacities, RESERVE)

	log.Info("Thresholds", "bottom", bottomT, "top", topT, "currentUse", previousCapacities[len(previousCapacities)-1])

	if previousCapacities[len(previousCapacities)-1] < bottomT {
		return false, false
	} else {
		if previousCapacities[len(previousCapacities)-1] > topT {
			return true, true
		}
		return true, false
	}
}
