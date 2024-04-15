package node

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var previousCapacities = []float64{}
var previousCpuReqUsage = []float64{}
var previousMemReqUsage = []float64{}
var previousGpuReqUsage = []float64{}

const (
	RESERVE   = 0.1
	NvidiaGPU = "nvidia.com/gpu"
)

type ResourcesUsage struct {
	CpuUsage float64
	MemUsage float64
	GpuUsage float64
}

// getNodeResource returns the capacity of the resource of a certain node
func getNodeResource(node *corev1.Node, resource corev1.ResourceName) float64 {
	capacity := 0.0

	for resourceName, allocatable := range node.Status.Allocatable {
		if resourceName == resource {
			capacity = allocatable.AsApproximateFloat64()
		}
	}
	return capacity
}

// getNodesPodUsage returns the current pod usage % of the nodes in the cluster
// not used in the current implementation
func getNodesPodUsage(ctx context.Context, client client.Client, log logr.Logger) (float64, error) {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods); err != nil {
		log.Error(err, "unable to list pods")
		return 0, err
	}

	nodes := &corev1.NodeList{}
	totalPodCapacity := 0
	if err := client.List(ctx, nodes); err != nil {
		log.Error(err, "unable to list nodes")
		return 0, err
	}
	for _, node := range nodes.Items {
		totalPodCapacity += int(getNodeResource(&node, corev1.ResourcePods))
	}

	return float64(len(pods.Items)) / float64(totalPodCapacity), nil
}

// getCurrentResourceRequests returns the current resource requests of all the pods in the cluster
func getCurrentResourceRequests(ctx context.Context, client client.Client, log logr.Logger) *ResourcesUsage {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods); err != nil {
		log.Error(err, "unable to list pods")
	}
	resources := &ResourcesUsage{
		CpuUsage: 0.0,
		MemUsage: 0.0,
		GpuUsage: 0.0,
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			q := container.Resources.Requests.Cpu()
			resources.CpuUsage += q.AsApproximateFloat64()
			q = container.Resources.Requests.Memory()
			resources.MemUsage += q.AsApproximateFloat64()
			//TODO add GPU compatibility
		}
	}
	return resources
}

// getNodesUsage returns the current resource usage % of the nodes in the cluster
func getNodesUsage(ctx context.Context, client client.Client, log logr.Logger) (currentResourceUsage *ResourcesUsage, err error) {
	nodes := &corev1.NodeList{}
	if err = client.List(ctx, nodes); err != nil {
		log.Error(err, "unable to list nodes")
		return &ResourcesUsage{}, err
	}
	allocatableCpu := 0.0
	allocatableMem := 0.0
	allocatableGpu := 0.0
	for _, node := range nodes.Items {
		allocatableCpu += getNodeResource(&node, corev1.ResourceCPU)
		allocatableMem += getNodeResource(&node, corev1.ResourceMemory)
		allocatableGpu += getNodeResource(&node, NvidiaGPU)
	}

	currentResourceUsage = getCurrentResourceRequests(ctx, client, log)
	if allocatableGpu != 0 {
		currentResourceUsage.GpuUsage /= allocatableGpu
	}
	currentResourceUsage.CpuUsage /= allocatableCpu
	currentResourceUsage.MemUsage /= allocatableMem

	return currentResourceUsage, nil
}

// isResourceFull checks if the resource is full and returns two booleans:
// - the first boolean is true if the resource is full (above the bottom threshold)
// - the second boolean is true if the resource is too full (above top threshold)
// The function also updates the previous usage
func isResourceFull(text string, previousUsage *[]float64, currentUsage float64, log logr.Logger) (bool, bool) {
	if len(*previousUsage) < 9 {
		*previousUsage = append(*previousUsage, currentUsage)
	} else {
		*previousUsage = append((*previousUsage)[1:], currentUsage)
	}
	bottom, top := getThresholds(*previousUsage, RESERVE)
	log.Info("Thresholds", "resource", text, "bottom", strconv.FormatFloat(bottom, 'f', -1, 64), "top", strconv.FormatFloat(top, 'f', -1, 64), "currentUse", strconv.FormatFloat((*previousUsage)[len(*previousUsage)-1], 'f', -1, 64))

	if (*previousUsage)[len(*previousUsage)-1] < bottom {
		return false, false
	} else {
		if (*previousUsage)[len(*previousUsage)-1] > top {
			return true, true
		}
		return true, false
	}
}

// getThresholds calculates the bottom and top thresholds for the resource usage from the previous usage
func getThresholds(previousUsage []float64, reserve float64) (float64, float64) {
	stdev, err := stats.StandardDeviation(previousUsage)

	if err != nil {
		stdev = 0
	}
	return 1 - (reserve + stdev), 1 - (reserve/3 + stdev)
}

// IsClusterBusy checks if the cluster is busy and returns two booleans:
// - the first boolean is true if the cluster is busy to start new jobs (above the bottom threshold)
// - the second boolean is true if the cluster is too busy and some jobs should interrupted (above top threshold)
func IsClusterBusy(ctx context.Context, client client.Client, log logr.Logger) (bool, bool) {
	currentResourcesUsage, err := getNodesUsage(ctx, client, log)
	if err != nil {
		log.Error(err, "unable to get nodes")
	}

	cpuStart, cpuKill := isResourceFull("CPU", &previousCpuReqUsage, currentResourcesUsage.CpuUsage, log)
	memStart, memKill := isResourceFull("MEM", &previousMemReqUsage, currentResourcesUsage.MemUsage, log)
	gpuStart, qpuKill := false, false
	if currentResourcesUsage.GpuUsage != 0 {
		gpuStart, qpuKill = isResourceFull("GPU", &previousGpuReqUsage, currentResourcesUsage.GpuUsage, log)
	}

	doNotStartMore := cpuStart || memStart || gpuStart
	startToKill := cpuKill || memKill || qpuKill

	return doNotStartMore, startToKill
}
