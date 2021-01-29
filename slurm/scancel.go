package main

import (
	"errors"

	jarvice "jarvice.io/core"
)

type SCancelCommand struct {
	Help  bool `short:"h" long:"help" description:"Show this help message"`
	Force bool `short:"f" description:"force job deletion"`
	Args  struct {
		JobNumber string `positional-arg-name:"number" description:"job number"`
	} `positional-args:"true" required:"1"`
}

var sCancelCommand SCancelCommand

func (x *SCancelCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return err
	}
	urlValues := cluster.GetUrlCreds()
	urlValues.Add("number", x.Args.JobNumber)
	api := "shutdown"
	if x.Force {
		api = "terminate"
	}
	if _, err := jarvice.ApiReq(cluster.Endpoint,
		api,
		urlValues); err == nil {

		return nil
	}
	return errors.New("scancel: HTTP error")
}

func init() {
	parser.AddCommand("scancel",
		"Slurm scancel",
		"Used to signal jobs or job steps that are under the control of Slurm",
		&sCancelCommand)
}
