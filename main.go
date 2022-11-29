package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	jarvice "jarvice.io/jarvice-hpc/core"
	logger "jarvice.io/jarvice-hpc/logger"
)

var parser = flags.NewNamedParser("jarvice", flags.PassDoubleDash|flags.IgnoreUnknown)

func printHelp(parser *flags.Parser) {
	// Print help for active command
	parser.Command = parser.Command.Active
	var b bytes.Buffer
	parser.WriteHelp(&b)
	fmt.Println(b.String())
}

func main() {
	var err error
	args := []string{}
	if args, err = jarvice.PreprocessArgs(os.Args); err != nil {
		logger.ErrorPrintf("flags error: %v",
			fmt.Errorf("PreprocessArg: %w", err))
		goto errHandler
	}
	if args, err = parser.ParseArgs(args); err != nil {
		logger.ErrorPrintf("flags error: %v",
			fmt.Errorf("ParseArgs: %w", err))
		goto errHandler
	}
	os.Exit(0)
errHandler:
	switch flagsErr := err.(type) {
	case *flags.Error:
		if flagsErr.Type == flags.ErrHelp ||
			flagsErr.Type == flags.ErrCommandRequired ||
			flagsErr.Type == flags.ErrRequired {
			logger.DebugPrintf("missing required flags")
			printHelp(parser)
			os.Exit(0)
		} else if flagsErr.Type == flags.ErrUnknownCommand {
			// HPC client CLI command that is not supported
			logger.DebugPrintf("%v not supported", args[0])
			if parser.Command.Active != nil {
				printHelp(parser)
			}
		} else if flagsErr.Type == flags.ErrMarshal {
			logger.DebugPrintf("Invalid syntax")
			printHelp(parser)
			os.Exit(1)
		}
		logger.DebugPrintf("unhandled flag error: %v", flagsErr.Error())
		os.Exit(1)
	case *jarvice.SgeError:
		logger.DebugPrintf("sge: %s", flagsErr.Error())
		fmt.Println(flagsErr.Error())
		os.Exit(1)
	default:
		// TODO create error type to prevent printing golang errors to user
		logger.DebugPrintf("main: unhandled error: %v", flagsErr.Error())
		os.Exit(1)

	}
}
