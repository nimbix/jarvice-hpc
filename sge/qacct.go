package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jarvice "jarvice.io/core"
)

type QAcctCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var qAcctCommand QAcctCommand

func qAcctPrintJob(number int, job jarvice.JarviceJob) {
	// Only print HPC jobs that have completed
	if job.EndTime == 0 || len(job.ApiSubmission.Queue) == 0 {
		//if job.EndTime == 0 {
		return
	}
	fmt.Println("==============================================================")
	fmt.Printf("qname\t%s\n", job.ApiSubmission.Queue)
	fmt.Printf("hostname\t%s\n", job.ApiSubmission.Machine.Type)
	fmt.Printf("group\t%s\n", job.User)
	fmt.Printf("owner\t%s\n", job.User)
	fmt.Printf("jobname\t%s\n", job.Label)
	fmt.Printf("jobnumber\t%d\n", number)
	fmt.Printf("account\t%s\n", job.User)
	// Times are best effort
	subTime := time.Unix(int64(job.SubmitTime), 0)
	fmt.Printf("qsub_time\t%s\n", subTime.Format(time.UnixDate))
	startTime := time.Unix(int64(job.StartTime), 0)
	fmt.Printf("start_time\t%s\n", startTime.Format(time.UnixDate))
	endTime := time.Unix(int64(job.EndTime), 0)
	fmt.Printf("end_time\t%s\n", endTime.Format(time.UnixDate))
	qAcctFailed := 0
	if job.ExitCode != 0 {
		qAcctFailed = 1
	}
	fmt.Printf("failed\t%d\n", qAcctFailed)
	fmt.Printf("exit_status\t%d\n", job.ExitCode)
}

func (x *QAcctCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	// Read JARVICE config for selected cluster
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return err
	}
	// need JARVICE API creds and 'completed' for /jarvice/jobs request
	urlValues := cluster.GetUrlCreds()
	urlValues.Add("completed", "true")
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"jobs",
		cluster.Insecure,
		urlValues); err == nil {
		var jarviceJobs jarvice.JarviceJobs
		if err := json.Unmarshal(resp, &jarviceJobs); err != nil {
			return errors.New("qacct: cannot read response")
		}
		for index, val := range jarviceJobs {
			qAcctPrintJob(index, val)
		}
		return nil
	}
	return errors.New("qacct: HTTP error")
}

func init() {
	parser.AddCommand("qacct",
		"SGE qacct",
		"report and account for Sun Grid Engine usage",
		&qAcctCommand)
}
