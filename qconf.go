package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	jarvice "jarvice.io/core"
)

type QConfCommand struct {
	Cluster string `short:"c" long:"cluster" description:"cluster name" default:"default"`
}

var qConfCommand QConfCommand

func (x *QConfCommand) Execute(args []string) error {
	config, err := jarvice.ReadJarviceConfig()
	if err != nil {
		return errors.New("qconf: cannot read JARVICE config")
	}
	clusterName := ""
	if !parser.Command.Active.FindOptionByLongName("cluster").IsSetDefault() {
		clusterName = x.Cluster
	} else {
		if val, err := jarvice.ReadJarviceConfigTarget(); err != nil {
			clusterName = x.Cluster
		} else {
			clusterName = val
		}
	}
	cluster := jarvice.JarviceCluster{}
	if val, ok := config[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("qconf: cannot find credentials for " + clusterName)
	}
	if resp, err := jarvice.ApiReq(cluster.Endpoint, "queues", cluster.GetUrlCreds()); err == nil {

		jarviceQueues := []string{}
		if err := json.Unmarshal([]byte(resp), &jarviceQueues); err != nil {
			return errors.New("qconf: cannot read response")
		}
		if len(jarviceQueues) < 1 {
			fmt.Println("default")
		} else {
			fmt.Printf("%s\n", strings.Join(jarviceQueues, "\n"))
		}
		return nil
	}
	return errors.New("qconf: HTTP error")
}

func init() {
	parser.AddCommand("qconf",
		"SGE qconf",
		"Sun Grid Engine Queue Configuration",
		&qConfCommand)
}
