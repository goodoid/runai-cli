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

	"github.com/kubeflow/arena/util"
	"github.com/kubeflow/arena/util/helm"
	"github.com/spf13/cobra"
	"time"
)

type PruneArgs struct {
	days  int64
	hours int64
}

func NewPruneCommand() *cobra.Command {
	var (
		pruneArgs PruneArgs
	)
	var command = &cobra.Command{
		Use:   "prune history job",
		Short: "prune history job",
		Run: func(cmd *cobra.Command, args []string) {
			util.SetLogLevel(logLevel)

			setupKubeconfig()
			client, err := initKubeClient()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			releaseMap, err := helm.ListReleaseMap()
			// log.Printf("releaseMap %v", releaseMap)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			// determine use cache
			useCache = true
			allPods, err = acquireAllPods(client)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			allJobs, err = acquireAllJobs(client)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			trainers := NewTrainers(client)
			jobs := []TrainingJob{}
			for name, ns := range releaseMap {
				for _, trainer := range trainers {
					if trainer.IsSupported(name, ns) {
						job, err := trainer.GetTrainingJob(name, ns)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						jobs = append(jobs, job)
						break
					}
				}
			}
			for _, job := range jobs {
				if GetJobRealStatus(job) != "RUNNING" {
					if job.Age() > (time.Duration(pruneArgs.days)*24*time.Hour + time.Duration(pruneArgs.hours)*time.Hour) {
						deleteTrainingJob(job.Name())
					}
				}
			}
		},
	}

	command.Flags().Int64Var(&pruneArgs.days, "day",  10, "Specify clean job's days.")
	command.Flags().Int64Var(&pruneArgs.hours, "hour", 0, "Specify clean job's hours.")
	return command
}
