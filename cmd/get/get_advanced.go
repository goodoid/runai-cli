// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package get

import (
	"fmt"
	"strings"

	tens "github.com/run-ai/runai-cli/cmd/tensorboard"
	"github.com/run-ai/runai-cli/cmd/trainer"
	"github.com/run-ai/runai-cli/pkg/types"
	"github.com/run-ai/runai-cli/pkg/util"
	"github.com/run-ai/runai-cli/pkg/util/kubectl"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

/*
* get App Configs by name, which is created by arena
 */
func GetTrainingTypes(name, namespace string, clientset kubernetes.Interface) (cms []string, err error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})
	if err != nil {
		return []string{}, err
	}
	cms = []string{}
	for _, trainingType := range trainer.KnownTrainingTypes {
		configName := fmt.Sprintf("%s-%s", name, trainingType)
		for _, configMap := range configMaps.Items {
			if configName == configMap.Name {
				cms = append(cms, trainingType)
			}
		}
	}

	return cms, nil
}

/*
* get App Configs by name, which is created by arena
 */
func getServingTypes(name, namespace string) (cms []string) {
	cms = []string{}
	for _, servingType := range trainer.KnownServingTypes {
		found := isTrainingConfigExist(name, servingType, namespace)
		if found {
			cms = append(cms, servingType)
		}
	}

	return cms
}

/**
*  check if the training config exist
 */
func isTrainingConfigExist(name, trainingType, namespace string) bool {
	configName := fmt.Sprintf("%s-%s", name, trainingType)
	return kubectl.CheckAppConfigMap(configName, namespace)
}

/**
* BuildTrainingJobInfo returns types.TrainingJobInfo
 */
func BuildJobInfo(job trainer.TrainingJob, clientset kubernetes.Interface) *types.JobInfo {

	tensorboard, err := tens.TensorboardURL(job.Name(), job.ChiefPod().Namespace, clientset)
	if tensorboard == "" || err != nil {
		log.Debugf("Tensorboard dones't show up because of %v, or tensorboard url %s", err, tensorboard)
	}

	instances := []types.Instance{}
	for _, pod := range job.AllPods() {
		isChief := false
		if pod.Name == job.ChiefPod().Name {
			isChief = true
		}

		instances = append(instances, types.Instance{
			Name:    pod.Name,
			Status:  strings.ToUpper(string(pod.Status.Phase)),
			Age:     util.ShortHumanDuration(job.Age()),
			Node:    pod.Status.HostIP,
			IsChief: isChief,
		})
	}

	return &types.JobInfo{
		Name:        job.Name(),
		Namespace:   job.Namespace(),
		Status:      types.JobStatus(GetJobRealStatus(job)),
		Duration:    util.ShortHumanDuration(job.Duration()),
		Trainer:     job.Trainer(),
		Priority:    getPriorityClass(job),
		Tensorboard: tensorboard,
		ChiefName:   job.ChiefPod().Name,
		Instances:   instances,
	}
}

/**
* getPriorityClass returns priority class name
 */
func getPriorityClass(job trainer.TrainingJob) string {
	pc := job.GetPriorityClass()
	if len(pc) == 0 {
		pc = "N/A"
	}

	return pc
}
