package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
		cluster.GetUrlCreds()); err == nil {

		var jarviceJobs jarvice.JarviceJobs
		if err := json.Unmarshal([]byte(resp), &jarviceJobs); err != nil {
			return errors.New("qstat: cannot read response")
		}
		retTable := [][]interface{}{
			{"job-ID", "prior", "name", "user", "state", "submit/start at", "queue"},
		}

		sgeState := "qw"
		for index, job := range jarviceJobs {
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
			retTable = append(retTable, []interface{}{index, "0", job.Label, job.User, sgeState, subTime.Format(time.UnixDate), job.ApiSubmission.Queue})
		}
		/*
			w := csv.NewWriter(os.Stdout)
			defer w.Flush()
			w.Comma = '\t'
			for _, record := range retTable {
				if err := w.Write(record); err != nil {
					return errors.New("qstat: cannot query queue")
				}
			}
		*/
		ii := 0
		for index, _ := range retTable {
			if index == 0 {
				fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", retTable[0][0], retTable[0][1], retTable[0][2], retTable[0][3], retTable[0][4], retTable[0][5], retTable[0][6])
				fmt.Printf("------------------------------------------------------------\n")
				continue
			}
			fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\t%s\n", retTable[index][0], retTable[index][1], retTable[index][2], retTable[index][3], retTable[index][4], retTable[index][5], retTable[index][6])
			ii += 1
		}
	}
	return nil
}

func init() {
	parser.AddCommand("qstat",
		"SGE qstat",
		"show the status of Sun Grid Engine jobs and queues",
		&qStatCommand)
}
