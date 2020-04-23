package commands

import (
	"strconv"
	"time"

	cmdTypes "github.com/kubeflow/arena/cmd/arena/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RunaiJob struct {
	*cmdTypes.BasicJobInfo
	trainerType       string
	chiefPod          *v1.Pod
	creationTimestamp metav1.Time
	interactive       bool
	createdByCLI      bool
	serviceUrls       []string
	deleted           bool
	podSpec           v1.PodSpec
	podMetadata       metav1.ObjectMeta
	jobMetadata       metav1.ObjectMeta
	namespace         string
	pods              []v1.Pod
}

const PodGroupNamePrefix = "pg-"

func NewRunaiJob(pods []v1.Pod, lastCreatedPod *v1.Pod, creationTimestamp metav1.Time, trainingType string, jobName string, createdByCLI bool, serviceUrls []string, deleted bool, podSpec v1.PodSpec, podMetadata metav1.ObjectMeta, jobMetadata metav1.ObjectMeta, namespace string, ownerResource cmdTypes.Resource) *RunaiJob {
	resources := append(cmdTypes.PodResources(pods), ownerResource)
	return &RunaiJob{
		pods:              pods,
		BasicJobInfo:      cmdTypes.NewBasicJobInfo(jobName, resources),
		chiefPod:          lastCreatedPod,
		creationTimestamp: creationTimestamp,
		trainerType:       trainingType,
		createdByCLI:      createdByCLI,
		serviceUrls:       serviceUrls,
		deleted:           deleted,
		podSpec:           podSpec,
		podMetadata:       podMetadata,
		jobMetadata:       jobMetadata,
		namespace:         namespace,
	}
}

// // Get the chief Pod of the Job.
func (rj *RunaiJob) ChiefPod() *v1.Pod {
	return rj.chiefPod
}

// Get the name of the Training Job
func (rj *RunaiJob) Name() string {
	return rj.BasicJobInfo.Name()
}

// Get the namespace of the Training Job
func (rj *RunaiJob) Namespace() string {
	return rj.namespace
}

// Get all the pods of the Training Job
func (rj *RunaiJob) AllPods() []v1.Pod {
	return rj.pods
}

// Get all the kubernetes resource of the Training Job
func (rj *RunaiJob) Resources() []cmdTypes.Resource {
	return rj.BasicJobInfo.Resources()
}

func (rj *RunaiJob) getStatus() v1.PodPhase {
	return rj.chiefPod.Status.Phase
}

// Get the Status of the Job: RUNNING, PENDING,
func (rj *RunaiJob) GetStatus() string {
	if value, exists := rj.jobMetadata.Annotations["unschedulable"]; exists {
		if value == "true" {
			return "Unschedulable"
		}
	}

	if rj.chiefPod == nil {
		return "Pending"
	}

	podStatus := rj.chiefPod.Status.Phase
	if rj.deleted {
		return "Terminating"
	}
	if podStatus == v1.PodPending {
		for _, containerStatus := range rj.chiefPod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				return containerStatus.State.Waiting.Reason
			}
		}
	}

	return string(podStatus)
}

// Return trainer Type, support MPI, standalone, tensorflow
func (rj *RunaiJob) Trainer() string {
	return rj.trainerType
}

// Get the Job Age
func (rj *RunaiJob) Age() time.Duration {
	if rj.creationTimestamp.IsZero() {
		return 0
	}
	return metav1.Now().Sub(rj.creationTimestamp.Time)
}

// TODO
// Get the Job Duration
func (rj *RunaiJob) Duration() time.Duration {
	if rj.chiefPod == nil {
		return 0
	}

	status := rj.getStatus()
	startTime := rj.StartTime()

	if startTime == nil {
		return 0
	}

	var finishTime metav1.Time = metav1.Now()

	if status == v1.PodSucceeded || status == v1.PodFailed {
		// The transition time of ready will be when the pod finished executing
		for _, condition := range rj.ChiefPod().Status.Conditions {
			if condition.Type == v1.PodReady {
				finishTime = condition.LastTransitionTime
			}
		}
	}

	return finishTime.Sub(startTime.Time)
}

func (rj *RunaiJob) CreatedByCLI() bool {
	return rj.createdByCLI
}

// Get start time
func (rj *RunaiJob) StartTime() *metav1.Time {
	if rj.chiefPod == nil {
		return nil
	}

	pod := rj.ChiefPod()
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodInitialized && condition.Status == v1.ConditionTrue {
			return &condition.LastTransitionTime
		}
	}

	return nil
}

// Get Dashboard
func (rj *RunaiJob) GetJobDashboards(client *kubernetes.Clientset) ([]string, error) {
	return []string{}, nil
}

// Requested GPU count of the Job
func (rj *RunaiJob) RequestedGPU() float64 {
	requestedGPUs := float64(0)
	for _, pod := range rj.pods {
		gpuFraction, GPUFractionErr := strconv.ParseFloat(pod.Annotations[runaiGPUFraction], 64)
		if GPUFractionErr == nil {
			requestedGPUs += gpuFraction
		}
	}

	if requestedGPUs != 0 {
		return requestedGPUs
	}

	val, ok := rj.podSpec.Containers[0].Resources.Limits[NVIDIAGPUResourceName]
	if !ok {
		return 0
	}

	return float64(val.Value())
}

// Requested GPU count of the Job
func (rj *RunaiJob) AllocatedGPU() float64 {
	if rj.chiefPod == nil {
		return 0
	}

	pod := rj.chiefPod

	if pod.Status.Phase == v1.PodRunning {
		return float64(rj.RequestedGPU())
	}

	return 0
}

// the host ip of the chief pod
func (rj *RunaiJob) HostIPOfChief() string {
	if rj.chiefPod == nil {
		return ""
	}

	// This will hold the node name even if not actually specified on pod spec by the user.
	// Copied from describe function of kubectl.
	// https://github.com/kubernetes/kubectl/blob/a20db94d5b5f052d991eaf29d626fb730b4886b7/pkg/describe/versioned/describe.go

	return rj.ChiefPod().Spec.NodeName
}

// The priority class name of the training job
func (rj *RunaiJob) GetPriorityClass() string {
	return ""
}

func (rj *RunaiJob) Image() string {
	return rj.podSpec.Containers[0].Image
}

func (rj *RunaiJob) Project() string {
	return rj.podMetadata.Labels["project"]
}

func (rj *RunaiJob) User() string {
	return rj.podMetadata.Labels["user"]
}

func (rj *RunaiJob) ServiceURLs() []string {
	return rj.serviceUrls
}

// IMPORTANT!!! This function is a duplication of GetPodGroupName in runai-scheduler repo.
// Do not make changes without changing it in runai-scheduler as well!
func (rj *RunaiJob) GetPodGroupName() string {
	pod := rj.chiefPod
	if pod == nil || pod.Spec.SchedulerName != SchedulerName {
		// This line is different in arena, it was added in order to get the podgroup of statefulsets with 0 replicas
		return PodGroupNamePrefix + rj.Name()
	}

	if jobName, found := pod.Labels["job-name"]; found && len(jobName) != 0 {
		return PodGroupNamePrefix + jobName
	}

	if pod.OwnerReferences == nil {
		return ""
	}

	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "StatefulSet" {
			return PodGroupNamePrefix + ownerRef.Name
		}
	}

	return ""
}
