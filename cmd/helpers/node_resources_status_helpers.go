package helpers

import (
	"github.com/run-ai/runai-cli/cmd/types"
)

type NodeResourcesStatusConvertor types.NodeResourcesStatus


func (c *NodeResourcesStatusConvertor) ToCpus() types.NodeCPUResource {
	nrs := (*types.NodeResourcesStatus)(c)
	return types.NodeCPUResource{
		Capacity:    int(nrs.Capacity.CPUs) / 1000,
		Allocatable: nrs.Allocatable.CPUs,
		Requested:   nrs.Requested.CPUs / 1000,
		Usage:       nrs.Usage.CPUs,
	}
}

func (c *NodeResourcesStatusConvertor) ToGpus() types.NodeGPUResource {
	nrs := (*types.NodeResourcesStatus)(c)
	return types.NodeGPUResource{
		Capacity:          int(nrs.Capacity.GPUs),
		Allocatable:       nrs.Allocatable.GPUs,
		Unhealthy:         int(nrs.Capacity.GPUs) - int(nrs.Allocatable.GPUs),
		AllocatedUnits:         nrs.AllocatedGPUsUnits,
		AllocatedFraction: nrs.Allocated.GPUs,
		Usage:             nrs.Usage.GPUs,
	}
}

func (c *NodeResourcesStatusConvertor) ToMemory() types.NodeMemoryResource {
	nrs := (*types.NodeResourcesStatus)(c)
	return types.NodeMemoryResource{
		Capacity:    nrs.Capacity.Memory,
		Allocatable: nrs.Allocatable.Memory,
		Requested:   nrs.Requested.Memory,
		Usage:       nrs.Usage.Memory,
	}
}

func (c *NodeResourcesStatusConvertor) ToGpuMemory() types.NodeMemoryResource {
	nrs := (*types.NodeResourcesStatus)(c)
	return types.NodeMemoryResource{
		Capacity:    nrs.Capacity.GPUMemory,
		Allocatable: nrs.Allocatable.GPUMemory,
		Usage:       nrs.Usage.GPUMemory,
	}
}

// todo: currently we are not understand enough the storage in kube
// func (nrs *NodeResourcesStatus) GetStorage() NodeStorageResource {
// 	return NodeStorageResource{
// 		Capacity:    c.Capacity.Storage,
// 		Allocatable: c.Allocatable.Storage,
// 		Allocated:   c.Allocatable.Storage,
// 		Limited:     c.Limited.Storage,
// 		Usage:       c.Usage.Storage,
// 		Requested:   c.Requested.Storage,
// 	}
// }