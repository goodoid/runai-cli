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

package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"io"

	"strconv"

	"github.com/kubeflow/arena/cmd/arena/commands/flags"
	cmdUtil "github.com/kubeflow/arena/cmd/arena/commands/util"
	"github.com/kubeflow/arena/pkg/client"
	"github.com/kubeflow/arena/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	var allNamespaces bool
	var command = &cobra.Command{
		Use:   "list",
		Short: "List all jobs.",
		Run: func(cmd *cobra.Command, args []string) {
			kubeClient, err := client.GetClient()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if err != nil {
				log.Errorf("Failed due to %v", err)
				os.Exit(1)
			}

			namespaceInfo, err := flags.GetNamespaceToUseFromProjectFlagIncludingAll(cmd, kubeClient, allNamespaces)

			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			cmdUtil.PrintShowingJobsInNamespaceMessage(namespaceInfo)

			jobs := []TrainingJob{}
			trainers := NewTrainers(kubeClient)
			for _, trainer := range trainers {
				if trainer.IsEnabled() {
					trainingJobs, err := trainer.ListTrainingJobs(namespaceInfo.Namespace)
					if err != nil {
						log.Errorf("Failed due to %v", err)
						os.Exit(1)
					}
					jobs = append(jobs, trainingJobs...)
				}
			}

			jobs = makeTrainingJobOrderdByAge(jobs)

			displayTrainingJobList(jobs, false)
		},
	}

	command.Flags().BoolVarP(&allNamespaces, "all-projects", "A", false, "list from all projects")

	return command
}

func displayTrainingJobList(jobInfoList []TrainingJob, displayGPU bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	labelField := []string{"NAME", "STATUS", "RUNNING PODS", "PENDING PODS", "AGE", "NODE", "IMAGE", "TYPE", "PROJECT", "USER", "GPUs", "CREATED BY CLI", "SERVICE URL(S)"}

	PrintLine(w, labelField...)

	for _, jobInfo := range jobInfoList {
		status := GetJobRealStatus(jobInfo)
		hostIP := jobInfo.HostIPOfChief()

		// For backward compatability. Indicat jobs on default namespace
		var projectName string
		if jobInfo.Namespace() == "default" {
			projectName = fmt.Sprintf("%s (old)", jobInfo.Project())
		} else {
			projectName = jobInfo.Project()
		}

		PrintLine(w, jobInfo.Name(),
			status,
			strconv.Itoa(int(jobInfo.RunningPods())), strconv.Itoa(int(jobInfo.PendingPods())),
			util.ShortHumanDuration(jobInfo.Age()),
			hostIP, jobInfo.Image(), jobInfo.Trainer(), projectName, jobInfo.User(), fmt.Sprintf("%g", jobInfo.RequestedGPU()), strconv.FormatBool(jobInfo.CreatedByCLI()), strings.Join(jobInfo.ServiceURLs(), ", "))
	}
	_ = w.Flush()
}

func PrintLine(w io.Writer, fields ...string) {
	//w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	buffer := strings.Join(fields, "\t")
	fmt.Fprintln(w, buffer)
}
