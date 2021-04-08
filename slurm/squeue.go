package main

import (
	"encoding/json"
	"errors"
	"strconv"

	jarvice "jarvice.io/core"
)

type SQueueCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
	// user JARVICE config credentials for API requests (ignore Username option)
	// Username string `short:"u" description:"username"`
}

var sQueueCommand SQueueCommand

func (x *SQueueCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	// use Cluster option name in query
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return err
	}
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"jobs",
		cluster.Insecure,
		cluster.GetUrlCreds()); err == nil {

		var jarviceJobs jarvice.JarviceJobs
		if err := json.Unmarshal([]byte(resp), &jarviceJobs); err != nil {
			return errors.New("squeue: cannot read response")
		}
		retTable := [][]string{
			{"JOBID", "PARTITION", "NAME", "USER", "ST", "TIME", "NODES", "NODELIST(REASON)"},
		}

		state := "PD"
		reason := "(Resources)"
		for index, job := range jarviceJobs {
			if len(job.ApiSubmission.Queue) == 0 {
				continue
			}
			switch job.Status {
			case "PROCESSING STARTING":
				state = "R"
				reason = job.ApiSubmission.Machine.Type
				/*
					case "SUBMITTED":
					case "COMPLETED":
					case "COMPLETED WITH ERROR":
					case "TERMINATED":
					case "CANCELED":
					case "EXEMPT":
					case "SEQUENTIALLY QUEUED":
				*/
			default:
				state = "PD"
			}
			jobScale := strconv.Itoa(job.ApiSubmission.Machine.Nodes)
			retTable = append(retTable, []string{strconv.Itoa(index),
				job.ApiSubmission.Queue,
				job.Label,
				job.User,
				state,
				"0:00",
				jobScale,
				reason})
		}
		jarvice.PrintTable(retTable, false)
	}
	return nil
}

func init() {
	parser.AddCommand("squeue",
		"Slurm squeue",
		"view information about jobs located in the Slurm scheduling queue",
		&sQueueCommand)
}
