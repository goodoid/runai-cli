package types

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/run-ai/runai-cli/cmd/util"
	prom "github.com/run-ai/runai-cli/pkg/prometheus"

	v1 "k8s.io/api/core/v1"
)

const (
	
	// prometheus query names
	TotalGpuMemoryPQ = "totalGpuMemory"
	UsedGpuMemoryPQ  = "usedGpuMemory"
	UsedCpuMemoryPQ  = "usedCpuMemory"
	UsedCpusPQ       = "usedCpus"
	UsedGpusPQ       = "usedGpus"
)

func NewNodeInfo(node v1.Node, pods []v1.Pod, promNodesMap prom.ItemsMap) NodeInfo {
	return NodeInfo{
		Node:           node,
		Pods:           pods,
		PrometheusNode: promNodesMap,
	}
}

type NodeInfo struct {
	Node           v1.Node
	Pods           []v1.Pod
	PrometheusNode prom.ItemsMap
}

func (ni *NodeInfo) GetStatus() NodeStatus {
	if !util.IsNodeReady(ni.Node) {
		return NodeNotReady
	}
	return NodeReady
}

func (ni *NodeInfo) GetGeneralInfo() NodeGeneralInfo {
	return NodeGeneralInfo{
		Name:      ni.Node.Name,
		Role:      strings.Join(util.GetNodeRoles(&ni.Node), ","),
		IPAddress: util.GetNodeInternalAddress(ni.Node),
		Status:    ni.GetStatus(),
	}
}

func (ni *NodeInfo) GetResourcesStatus() NodeResourcesStatus {

	nodeResStatus := NodeResourcesStatus{}
	podResStatus := PodResourcesStatus{}

	for _, pod := range ni.Pods {
		podResStatus.Add(GetPodResourceStatus(pod))
	}

	// adding the kube data
	nodeResStatus.Requested = podResStatus.Requested
	nodeResStatus.Allocated = podResStatus.Requested
	nodeResStatus.Allocated.GPUs = podResStatus.Allocated.GPUs
	nodeResStatus.Limited = podResStatus.Limited
	
	nodeResStatus.Capacity.AddKubeResourceList(ni.Node.Status.Capacity)
	// fix the gpus capacity (when there is a job that using fractional gpu the gpu will not appear in the node > status > capacity so we need to override the capacity.gpus  )
	totalGpus := int(util.AllocatableGpuInNode(ni.Node))
	// check that the totalGpues is set
	if (totalGpus > int(nodeResStatus.Capacity.GPUs)) {
		nodeResStatus.FractionalAllocatedGpuUnits = len(util.GetSharedGPUsIndexUsedInPods(ni.Pods))
		nodeResStatus.Capacity.GPUs = float64(totalGpus)
		// update the allocatable too
		nodeResStatus.Allocatable.GPUs += float64(nodeResStatus.FractionalAllocatedGpuUnits)
	}

	nodeResStatus.Allocatable.AddKubeResourceList(ni.Node.Status.Allocatable)
	nodeResStatus.AllocatedGPUsUnits = nodeResStatus.FractionalAllocatedGpuUnits + int(podResStatus.Limited.GPUs)

	// adding the prometheus data
	p, ok := ni.PrometheusNode[ni.Node.Name]
	if ok {
		// set usages
		err := hasError(
			setFloatPromData(&nodeResStatus.Usage.CPUs, p, UsedCpusPQ),
			setFloatPromData(&nodeResStatus.Usage.GPUs, p, UsedGpusPQ),
			setFloatPromData(&nodeResStatus.Usage.Memory, p, UsedCpuMemoryPQ),
			setFloatPromData(&nodeResStatus.Usage.GPUMemory, p, UsedGpuMemoryPQ),
			// setFloatPromData(&nodeResStatus.Usage.Storage, p, UsedStoragePQ)

			// set total
			setFloatPromData(&nodeResStatus.Capacity.GPUMemory, p, TotalGpuMemoryPQ),
		)

		if err != nil {
			log.Debugf("Failed to extract prometheus data, %v",err)
		}
	}

	return nodeResStatus
}

func (nodeInfo *NodeInfo) IsGPUExclusiveNode() bool {
	value, ok := nodeInfo.Node.Status.Allocatable[NVIDIAGPUResourceName]

	if ok {
		ok = (int(value.Value()) > 0)
	}

	return ok
}
