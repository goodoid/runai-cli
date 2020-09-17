package util


import (
	"fmt"
	// "os"
	"time"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	// cmdutil "k8s.io/kubectl/pkg/cmd/util"

)

const (
	NotReadyPodTimeoutMsg = "Timeout waiting for job to start running"
)

// WaitForPod waiting to the pod phase to become running
func WaitForPod(getPod func() (*v1.Pod, error), timeout time.Duration, timeoutMsg string, exitCondition func(*v1.Pod, int) (bool, error) ) ( pod *v1.Pod, err error)  {
	shouldStopAt := time.Now().Add( timeout)

	for i, exit := 0, false;; i++ {
		pod, err = getPod()
		if err != nil {
			return 
		}

		exit, err = exitCondition(pod, i)
		if err != nil || exit {
			return 
		}

		if shouldStopAt.Before( time.Now()) {
			return nil, fmt.Errorf(timeoutMsg)
		}
		time.Sleep(time.Second)	
	}
}

// PodRunning check if the pod is running and ready
func PodRunning(pod *v1.Pod, i int) (exit bool, err error) {
	phase := pod.Status.Phase

	switch phase {
	case v1.PodPending:
		break
	case v1.PodRunning:
		conditions := pod.Status.Conditions
		if conditions == nil {
			return false, nil
		}
		for i := range conditions {
			if conditions[i].Type == corev1.PodReady &&
				conditions[i].Status == corev1.ConditionTrue {
					exit = true 
			}
		}
		
	default:
		err = fmt.Errorf("Can't connect to the pod: %s in phase: %s",pod.Name, phase)
	}

	if i == 0 && !exit && err == nil{
		fmt.Println("Waiting for pod to start running...")
	} 

	return
}

