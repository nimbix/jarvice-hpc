package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	jarvice "jarvice.io/core"
)

type QStatCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
	// user JARVICE config credentials for API requests (ignore Username option)
	// Username string `short:"u" description:"username"`
	Cluster string `short:"c" long:"cluster" description:"cluster name" default:"default"`
}

var qStatCommand QStatCommand

func sgeVersion() {
	fmt.Println("SGE 8.1.9")
	return
}

func (x *QStatCommand) Execute(args []string) error {
	if x.Help {
		// return version string w/ normal exit
		sgeVersion()
		return nil
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
			return errors.New("qstat: cannot read response")
		}
		retTable := [][]string{
			{"job-ID", "prior", "name", "user", "state", "submit/start at", "queue"},
		}

		sgeState := "qw"
		for index, job := range jarviceJobs {
			// Only show jobs from queue
			if len(job.ApiSubmission.Queue) == 0 {
				continue
			}
			switch job.Status {
			case "PROCESSING STARTING":
				sgeState = "r"
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
				sgeState = "qw"
			}
			subTime := time.Unix(int64(job.SubmitTime), 0)
			retTable = append(retTable, []string{strconv.Itoa(index),
				"0",
				job.Label,
				job.User,
				sgeState,
				subTime.Format(time.UnixDate),
				job.ApiSubmission.Queue})
		}
		jarvice.PrintTable(retTable, true)
	}
	return nil
}

func init() {
	parser.AddCommand("qstat",
		"SGE qstat",
		"show the status of Sun Grid Engine jobs and queues",
		&qStatCommand)
}
