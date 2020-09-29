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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/run-ai/runai-cli/cmd/constants"
	"github.com/run-ai/runai-cli/cmd/trainer"
	"github.com/run-ai/runai-cli/pkg/client"
	"github.com/run-ai/runai-cli/pkg/util"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/run-ai/runai-cli/cmd/flags"
	cmdUtil "github.com/run-ai/runai-cli/cmd/util"
	"github.com/run-ai/runai-cli/pkg/config"
	cmdTypes "github.com/run-ai/runai-cli/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type eventAndName struct {
	event v1.Event
	name  string
	index int
}

var (
	output       string
	dashboardURL string
)

// NewGetCommand
func NewGetCommand() *cobra.Command {
	printArgs := PrintArgs{}
	var command = &cobra.Command{
		Use:   "get JOB_NAME",
		Short: "Display details of a job.",
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			name := args[0]

			kubeClient, err := client.GetClient()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			clientset := kubeClient.GetClientset()
			namespace, err := flags.GetNamespaceToUseFromProjectFlag(cmd, kubeClient)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			job, err := SearchTrainingJob(kubeClient, name, "", namespace)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			printTrainingJob(clientset, job, printArgs)
		},
	}

	command.Flags().BoolVarP(&printArgs.ShowEvents, "events", "e", true, "Show events relating to job lifecycle.")
	command.Flags().StringVarP(&printArgs.Output, "output", "o", "", "Output format. One of: json|yaml|wide")

	command.Flags().MarkDeprecated("events", "default is true")
	return command
}

type PrintArgs struct {
	ShowEvents bool
	Output     string
}

/*
* search the training job with name and training type
 */
func SearchTrainingJob(kubeClient *client.Client, jobName string, trainingType string, namespaceInfo cmdTypes.NamespaceInfo) (job trainer.TrainingJob, err error) {
	if len(trainingType) > 0 {
		if trainer.IsKnownTrainingType(trainingType) {
			job, err = getTrainingJobByType(kubeClient, jobName, namespaceInfo.Namespace, trainingType)
			if err != nil {
				if isTrainingConfigExist(jobName, trainingType, namespaceInfo.Namespace) {
					log.Warningf("Failed to get the training job %s, but the trainer config is found, please clean it by using '%s delete %s --type %s'.",
						jobName,
						config.CLIName,
						jobName,
						trainingType)
				}
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("%s is unknown training type, please choose a known type from %v",
				trainingType,
				trainer.KnownTrainingTypes)
		}
	} else {
		jobs, errorGetByName := getTrainingJobsByName(kubeClient, jobName, namespaceInfo)
		if errorGetByName != nil {
			traningTypes, err := GetTrainingTypes(jobName, namespaceInfo.Namespace, kubeClient.GetClientset())
			if err != nil {
				return nil, err
			}
			if len(traningTypes) > 0 {
				log.Warningf("Failed to get the training job %s, but the trainer config is found, please clean it by using '%s delete %s'.",
					jobName,
					config.CLIName,
					jobName)
			}
			return nil, errorGetByName
		}

		if len(jobs) > 1 {
			return nil, fmt.Errorf("There are more than 1 training jobs with the same name %s, please check it with `%s list | grep %s`",
				jobName,
				config.CLIName,
				jobName)
		} else {
			job = jobs[0]
		}
	}

	return job, nil
}

func getTrainingJob(kubeClient *client.Client, name, namespace string) (job trainer.TrainingJob, err error) {
	// trainers := NewTrainers(client, )

	trainers := trainer.NewTrainers(kubeClient)
	for _, trainer := range trainers {
		if trainer.IsSupported(name, namespace) {
			return trainer.GetTrainingJob(name, namespace)
		} else {
			log.Debugf("the job %s in namespace %s is not supported by %v", name, namespace, trainer.Type())
		}
	}

	return nil, fmt.Errorf("Failed to find the training job %s in namespace %s", name, namespace)
}

func getTrainingJobByType(kubeClient *client.Client, name, namespace, trainingType string) (job trainer.TrainingJob, err error) {
	// trainers := NewTrainers(client, )

	trainers := trainer.NewTrainers(kubeClient)
	for _, trainer := range trainers {
		if trainer.Type() == trainingType {
			return trainer.GetTrainingJob(name, namespace)
		} else {
			log.Debugf("the job %s with type %s in namespace %s is not expected type %v",
				name,
				trainer.Type(),
				namespace,
				trainingType)
		}
	}

	return nil, fmt.Errorf("Failed to find the training job %s in namespace %s", name, namespace)
}

func getTrainingJobsByName(kubeClient *client.Client, name string, namespaceInfo cmdTypes.NamespaceInfo) (jobs []trainer.TrainingJob, err error) {
	jobs = []trainer.TrainingJob{}
	trainers := trainer.NewTrainers(kubeClient)
	for _, trainer := range trainers {
		if trainer.IsSupported(name, namespaceInfo.Namespace) {
			job, err := trainer.GetTrainingJob(name, namespaceInfo.Namespace)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, job)
		} else {
			log.Debugf("the job %s in namespace %s is not supported by %v", name, namespaceInfo.Namespace, trainer.Type())
		}
	}

	if len(jobs) == 0 {
		log.Debugf("Failed to find the training job %s in namespace %s", name, namespaceInfo.Namespace)
		return nil, cmdUtil.GetJobDoesNotExistsInNamespaceError(name, namespaceInfo)
	}

	return jobs, nil
}

func printTrainingJob(client kubernetes.Interface, job trainer.TrainingJob, printArgs PrintArgs) {
	switch printArgs.Output {
	case "name":
		fmt.Println(job.Name())
		// for future CRD support
	case "json":
		outBytes, err := json.MarshalIndent(BuildJobInfo(job, client), "", "    ")
		if err != nil {
			fmt.Printf("Failed due to %v", err)
		} else {
			fmt.Println(string(outBytes))
		}
	case "yaml":
		outBytes, err := yaml.Marshal(BuildJobInfo(job, client))
		if err != nil {
			fmt.Printf("Failed due to %v", err)
		} else {
			fmt.Println(string(outBytes))
		}
	case "wide", "":
		printSingleJobHelper(client, job, printArgs)
	default:
		log.Fatalf("Unknown output format: %s", printArgs.Output)
	}
}

func printSingleJobHelper(client kubernetes.Interface, job trainer.TrainingJob, printArgs PrintArgs) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	printJobSummary(w, job)

	// apply a dummy FgDefault format to align tabwriter with the rest of the columns
	fmt.Fprintf(w, "Pods:\n")
	fmt.Fprintf(w, "POD\tSTATUS\tTYPE\tAGE\tNODE\n")
	pods := job.AllPods()

	for _, pod := range pods {
		// hostIP := "N/A"

		var hostIP string
		if pod.Spec.NodeName != "" {
			hostIP = pod.Spec.NodeName + "/" + pod.Status.HostIP
		} else {
			hostIP = pod.Status.HostIP
		}
		// if pod.Status.Phase == v1.PodRunning {
		// }

		if len(hostIP) == 0 {
			hostIP = "N/A"
		}

		podStatus, ok := pod.Annotations[constants.WorkloadCalculatedStatus]
		if !ok {
			podStatus = string(pod.Status.Phase)
		}
		var podCreationTime time.Duration
		if pod.CreationTimestamp.IsZero() {
			podCreationTime = 0
		}

		podCreationTime = metav1.Now().Sub(pod.CreationTimestamp.Time)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", pod.Name,
			strings.ToUpper(podStatus),
			strings.ToUpper(job.Trainer()),
			util.ShortHumanDuration(podCreationTime),
			hostIP)
	}

	if printArgs.ShowEvents {
		printEvents(client, w, job.Namespace(), job)
	}

	_ = w.Flush()

}

func printJobSummary(w io.Writer, job trainer.TrainingJob) {
	fmt.Fprintf(w, "NAME: %s\n", job.Name())
	fmt.Fprintf(w, "NAMESPACE: %s\n", job.Namespace())
	fmt.Fprintf(w, "TYPE: %s\n", job.Trainer())
	fmt.Fprintf(w, "STATUS: %s\n", GetJobRealStatus(job))
	fmt.Fprintf(w, "TRAINING DURATION: %s\n", util.ShortHumanDuration(job.Duration()))
	fmt.Fprintf(w, "GPUS: %g\n", job.RequestedGPU())
	fmt.Fprintf(w, "TOTAL REQUESTED GPUS: %g\n", job.TotalRequestedGPUs())
	fmt.Fprintf(w, "ALLOCATED GPUS: %g\n", job.CurrentAllocatedGPUs())
	fmt.Fprintf(w, "RUNNING PODS: %d\n", job.RunningPods())
	fmt.Fprintf(w, "PENDING PODS: %d\n", job.PendingPods())
	fmt.Fprintf(w, "PARALLELISM: %d\n", job.Parallelism())
	fmt.Fprintf(w, "COMPLETIONS: %d\n", job.Completions())
	fmt.Fprintf(w, "SUCCEEDED PODS: %d\n", job.Succeeded())
	fmt.Fprintf(w, "FAILED PODS: %d\n", job.Failed())
	fmt.Fprintf(w, "IS DISTRIBUTED WORKLOAD: %s\n", strconv.FormatBool(job.WorkloadType() == "MPIJob"))
	fmt.Fprintf(w, "CREATED BY CLI: %s\n", strconv.FormatBool(job.CreatedByCLI()))
	fmt.Fprintf(w, "SERVICE URL(S): %s\n", strings.Join(job.ServiceURLs(), ", "))
	fmt.Fprintln(w, "")

}

func printEvents(clientset kubernetes.Interface, w io.Writer, namespace string, job trainer.TrainingJob) {
	fmt.Fprintf(w, "\nEvents: \n")
	eventsMap, err := getResourcesEvents(clientset, namespace, job)
	if err != nil {
		fmt.Fprintf(w, "Get job events failed, due to: %v", err)
		return
	}
	if len(eventsMap) == 0 {
		fmt.Fprintln(w, "No events for resources")
		return
	}
	fmt.Fprintf(w, "SOURCE\tTYPE\tAGE\tMESSAGE\n")
	fmt.Fprintf(w, "--------\t----\t---\t-------\n")

	for _, eventAndName := range eventsMap {
		instanceName := fmt.Sprintf("%s/%s", strings.ToLower(eventAndName.event.InvolvedObject.Kind), eventAndName.name)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
			instanceName,
			eventAndName.event.Type,
			util.ShortHumanDuration(time.Now().Sub(eventAndName.event.CreationTimestamp.Time)),
			fmt.Sprintf("[%s] %s", eventAndName.event.Reason, eventAndName.event.Message))
		// empty line for per pod
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "", "", "", "", "", "")
	}
}

// Get real job status
// WHen has pods being pending, tfJob still show in Running state, it should be Pending
func GetJobRealStatus(job trainer.TrainingJob) string {
	hasPendingPod := false
	jobStatus := job.GetStatus()
	if jobStatus == "RUNNING" {
		pods := job.AllPods()
		for _, pod := range pods {
			if pod.Status.Phase == v1.PodPending {
				log.Debugf("pod %s is pending", pod.Name)
				hasPendingPod = true
				break
			}
		}
		if hasPendingPod {
			jobStatus = "PENDING"
		}
	}
	return jobStatus
}

// Get Event of the Job
func getResourcesEvents(client kubernetes.Interface, namespace string, job trainer.TrainingJob) ([]eventAndName, error) {
	events, err := client.CoreV1().Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return []eventAndName{}, err
	}

	return getSortedEvents(events.Items, job.Resources(), job.GetPodGroupName()), nil
}

func getSortedEvents(items []v1.Event, resources []cmdTypes.Resource, podGroupName string) []eventAndName {
	eventAndNames := []eventAndName{}
	index := 0
	for _, event := range items {
		for _, resource := range resources {
			if event.InvolvedObject.Kind == string(resource.ResourceType) && string(event.InvolvedObject.UID) == resource.Uid {
				eventAndNames = append(eventAndNames, eventAndName{event, resource.Name, index})
				index++
				break
			}
		}

		// TODO: We should add pogGroup as a resource of a job and remove this part.
		if len(podGroupName) > 0 && event.InvolvedObject.Name == podGroupName {
			eventAndNames = append(eventAndNames, eventAndName{event, podGroupName, index})
			index++
		}

	}

	sort.Slice(eventAndNames, func(i, j int) bool {
		lv := eventAndNames[i]
		rv := eventAndNames[j]
		if lv.event.CreationTimestamp.Time.Before(rv.event.CreationTimestamp.Time) {
			return true
		}

		if lv.event.CreationTimestamp.Time.After(rv.event.CreationTimestamp.Time) {
			return false

		}

		return lv.index < rv.index
	})

	return eventAndNames
}