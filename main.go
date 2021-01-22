package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
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
	cmdArgs := os.Args
	// strip path for arg 0
	cmdArgs[0] = filepath.Base(os.Args[0])
	if args, err := parser.ParseArgs(cmdArgs); err != nil {
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
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}
