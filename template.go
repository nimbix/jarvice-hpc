package main

import (
	jarvice "jarvice.io/jarvice-hpc/core"
	logger "jarvice.io/jarvice-hpc/logger"
)

type TemplateCommand struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var templateCommand TemplateCommand

func (x *TemplateCommand) Execute(args []string) error {
	if x.Help {
		return jarvice.CreateHelpErr()
	}
	msg := "Hello World"
	// DEBUG message
	logger.DebugPrintf(msg)
	// INFO message
	logger.InfoPrintf(msg)
	// WARNING message
	logger.WarningPrintf(msg)
	// ERROR message
	logger.ErrorPrintf(msg)
	// CRITICAL message
	logger.CriticalPrintf(msg)
	return nil
}

func init() {
	parser.AddCommand("template",
		"short description",
		"long description",
		&templateCommand)
}
