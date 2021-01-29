package main

import (
	"fmt"

	jarvice "jarvice.io/core"
)

type TemplateCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var templateCommand TemplateCommand

func (x *TemplateCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	fmt.Println("Hello World")
	return nil
}

func init() {
	parser.AddCommand("template",
		"short description",
		"long description",
		&templateCommand)
}
