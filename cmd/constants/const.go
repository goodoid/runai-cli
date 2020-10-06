package constants

const (
	RUNAI_QUEUE_LABEL = "runai/queue"
)

// Same statuses appear in the scheduler - update both if needed
var Status = struct {
	Running   string
	Pending   string
	Succeeded string
	Deleted   string
	Failed    string
	TimedOut  string
	Preempted string
	Unknown   string
}{
	Running:   "Running",
	Pending:   "Pending",
	Succeeded: "Succeeded",
	Deleted:   "Deleted",
	Failed:    "Failed",
	TimedOut:  "TimedOut",
	Preempted: "Preempted",
	Unknown:   "Unknown",
}

// todo organize

const (
	// remove from here to the gpu const
	runaiGPUFraction = "gpu-fraction"
	runaiGPUIndex    = "runai-gpu"

	PodGroupAnnotationForPod = "pod-group-name"

	CHART_PKG_LOC = "CHARTREPO"
	// GPUResourceName is the extended name of the GPU resource since v1.8
	// this uses the device plugin mechanism
	NVIDIAGPUResourceName = "nvidia.com/gpu"
	ALIYUNGPUResourceName = "aliyun.com/gpu-mem"

	DeprecatedNVIDIAGPUResourceName = "alpha.kubernetes.io/nvidia-gpu"

	SchedulerName = "runai-scheduler"

	masterLabelRole = "node-role.kubernetes.io/master"

	gangSchdName = "kube-batchd"

	// labelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	labelNodeRolePrefix = "node-role.kubernetes.io/"

	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"

	WorkloadCalculatedStatus     = "runai-calculated-status"
	WorkloadRunningPods          = "runai-running-pods"
	WorkloadPendingPods          = "runai-pending-pods"
	WorkloadUsedNodes            = "runai-used-nodes"
	PodGroupRequestedGPUs        = "runai-podgroup-requested-gpus"
	WorkloadCurrentAllocatedGPUs = "runai-current-allocated-gpus"
	WorkloadCurrentRequestedGPUs = "runai-current-requested-gpus"
	WorkloadTotalRequestedGPUs   = "runai-total-requested-gpus"
	AliyunENIAnnotation          = "k8s.aliyun.com/eni"
)

