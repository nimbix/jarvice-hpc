package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	jarvice "jarvice.io/jarvice-hpc/core"
)

type QConfCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var qConfCommand QConfCommand

func (x *QConfCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return &jarvice.SgeError {
			Command: "qconf",
			Err: err,
		}
	}
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"queues",
		cluster.Insecure,
		cluster.GetUrlCreds()); err == nil {

		jarviceQueues := []string{}
		if err := json.Unmarshal([]byte(resp), &jarviceQueues); err != nil {
			return &jarvice.SgeError {
				Command: "qconf",
				Err: errors.New("cannot read response"),
			}
		}
		if len(jarviceQueues) < 1 {
			fmt.Println("default")
		} else {
			fmt.Printf("%s\n", strings.Join(jarviceQueues, "\n"))
		}
		return nil
	}
	return &jarvice.SgeError {
		Command: "qconf",
		Err: errors.New("HTTP error"),
	}
}

func init() {
	parser.AddCommand("qconf",
		"SGE qconf",
		"Sun Grid Engine Queue Configuration",
		&qConfCommand)
}
