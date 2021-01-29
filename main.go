package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	jarvice "jarvice.io/core"
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
		goto errHandler
	}
	if args, err = parser.ParseArgs(args); err != nil {
		goto errHandler
	}
	os.Exit(0)
errHandler:
	switch flagsErr := err.(type) {
	case *flags.Error:
		if flagsErr.Type == flags.ErrHelp ||
			flagsErr.Type == flags.ErrCommandRequired ||
			flagsErr.Type == flags.ErrRequired {
			printHelp(parser)
			os.Exit(0)
		} else if flagsErr.Type == flags.ErrUnknownCommand {
			// HPC client CLI command that is not supported
			fmt.Printf("`%v' not supported\n\n\n", args[0])
			if parser.Command.Active != nil {
				printHelp(parser)
			}
		} else if flagsErr.Type == flags.ErrMarshal {
			fmt.Println("\n\nInvalid syntax\n\n")
			printHelp(parser)
			os.Exit(1)
		}
		fmt.Println(flagsErr.Error())
		os.Exit(1)

	default:
		// TODO create error type to prevent printing golang errors to user
		fmt.Println(flagsErr.Error())
		os.Exit(1)

	}
}
