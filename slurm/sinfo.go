package main

import (
	"encoding/json"
	"errors"
	"strconv"

	jarvice "jarvice.io/core"
)

type SInfoCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var sInfoCommand SInfoCommand

func printPartitionInfo(queues jarvice.JarviceQueues) {
	table := [][]string{
		{"PARTITION", "AVAIL", "TIMELIMIT", "NODES", "STATE", "NODELIST"},
	}
	for _, queue := range queues {
		scaleString := strconv.Itoa(queue.MachineScale)
		table = append(table, []string{
			queue.Name,
			"up",
			"infinite",
			scaleString,
			"idle",
			queue.DefaultMachine + "[0-" + strconv.Itoa(queue.MachineScale-1) + "]"})
	}
	jarvice.PrintTable(table, false)
}

func (x *SInfoCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return err
	}
	urlValues := cluster.GetUrlCreds()
	urlValues.Add("info", "true")
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"queues",
		cluster.Insecure,
		urlValues); err == nil {

		jarviceQueues := jarvice.JarviceQueues{}
		if err := json.Unmarshal([]byte(resp), &jarviceQueues); err != nil {
			return errors.New("sinfo: cannot read response")
		}
		printPartitionInfo(jarviceQueues)
		return nil
	}
	return errors.New("sinfo: HTTP error")
}

func init() {
	parser.AddCommand("sinfo",
		"Slurm sinfo",
		"View information about Slurm nodes and partitions.",
		&sInfoCommand)
}
