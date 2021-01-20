package sge

import (
	"errors"
	"flag"
	"strconv"
	"strings"

	core "jarvice.io/jarvice-hpc/core"
)

type JobScript = core.JobScript

// Option descriptions
const ()

// List of support options
// map[string]struct{} enables querying supported options using:
// _, ok := sBatchSupportedArgs()["<option>"]
func qStatSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"help": struct{}{},
		"u":    struct{}{},
	}
}

type sgeOptions = map[string]interface{}

func qAcctSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"j": struct{}{},
		"o": struct{}{},
		"b": struct{}{},
	}
}

func parseQAcctArgs(args []string) (options sgeOptions, flags *flag.FlagSet, err error) {

	flags = flag.NewFlagSet("qacct", flag.ContinueOnError)

	options = make(sgeOptions)
	options["j"] = flags.String("j", "", "job id")
	options["o"] = flags.String("o", "", "owner")
	options["b"] = flags.String("b", "", "begin time")
	if flags.Parse(args) != nil {
		err = errors.New("qacct: cannot process flags")
		return
	}
	return
}

func parseQStatArgs(args []string) (options sgeOptions, flags *flag.FlagSet, err error) {

	flags = flag.NewFlagSet("qstat", flag.ContinueOnError)

	options = make(sgeOptions)
	options["help"] = flags.Bool("help", false, "help")
	options["u"] = flags.String("u", "", "username")
	if flags.Parse(args) != nil {
		err = errors.New("qstat: cannot process flags")
		return
	}

	return
}

func qConfSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"sql": struct{}{},
	}
}

func parseQConfArgs(args []string) (options sgeOptions, flags *flag.FlagSet, err error) {

	flags = flag.NewFlagSet("qconf", flag.ContinueOnError)

	options = make(sgeOptions)
	options["sql"] = flags.Bool("sql", false, "show queue list")

	if flags.Parse(args) != nil {
		err = errors.New("qconf: cannot process flags")
		return
	}

	return
}

func qSubSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"b":    struct{}{},
		"S":    struct{}{},
		"N":    struct{}{},
		"cwd":  struct{}{},
		"q":    struct{}{},
		"l":    struct{}{},
		"j":    struct{}{},
		"pe":   struct{}{},
		"hard": struct{}{},
		"soft": struct{}{},
	}
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, " ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *arrayFlags) Get() interface{} {
	return i
}

func validatePeScale(str string) (ret string, err error) {
	if i, e := strconv.ParseInt(str, 10, 64); e == nil {
		if i > 0 {
			ret = str
			return
		}
	}
	err = errors.New("pe scale must be positive integer: -pe NAME INT")
	return
}

func parseQSubArgs(args []string) (retOptions sgeOptions, jobScriptFilename, submitCommand string, err error) {

	retOptions = make(sgeOptions)
	for index, val := range args {
		if val == "-soft" {
			args[index] = "ARGsoft"
		} else if val == "-hard" {
			args[index] = "ARGhard"
		} else if val == "-pe" {
			if len(args) > index+2 {
				if scale, e := validatePeScale(args[index+2]); e != nil {
					err = e
					return
				} else {
					args[index+2] = args[index+1] + "=" + scale
					args = append(args[:index+1], args[index+2:]...)
				}
			} else {
				err = errors.New("invalid syntax: -pe NAME INT")
				return
			}
		}
	}
	var mySoftRes arrayFlags
	var myHardRes arrayFlags
	loop := true
	var options sgeOptions
	for loop {
		options = make(sgeOptions)
		flags := flag.NewFlagSet("", flag.ContinueOnError)
		options["b"] = flags.String("b", "", "Command binary/script y/n")
		options["S"] = flags.String("S", "/bin/sh", "Shell")
		options["N"] = flags.String("N", "", "Job name")
		options["cwd"] = flags.Bool("cwd", false, "Current working directory")
		options["q"] = flags.String("q", "", "Submit queue")
		options["pe"] = flags.String("pe", "", "Parallel environment")
		// TODO: not implemented
		options["V"] = flags.Bool("V", false, "")
		options["o"] = flags.String("o", "", "")
		options["e"] = flags.String("e", "", "")
		options["m"] = flags.String("m", "", "")
		options["j"] = flags.String("j", "", "")
		if len(args) > 1 {
			if args[0] == "ARGsoft" {
				flags.Var(&mySoftRes, "l", "soft resource")
				flags.Parse(args[1:])
			} else if args[0] == "ARGhard" {
				flags.Var(&myHardRes, "l", "hard resource")
				flags.Parse(args[1:])
			} else {
				flags.Var(&myHardRes, "l", "hard resource")
				flags.Parse(args)
			}
		} else {
			if flags.Parse(args) != nil {
				flags.PrintDefaults()
				return
			}

		}
		flags.Visit(func(f *flag.Flag) {
			retOptions[f.Name] = f.Value.(flag.Getter).Get()
		})
		args = flags.Args()
		if flags.NFlag() == 0 {
			loop = false
			if num := flags.NArg(); num == 0 {
				jobScriptFilename = "STDIN"
			} else if num == 1 {
				jobScriptFilename = args[0]
			} else {
				jobScriptFilename = flags.Args()[0]
				submitCommand = jobScriptFilename
				if submitCommand == "--" {
					submitCommand = strings.Join(flags.Args()[1:], " ")
				}
			}
		}
		// XXX
	}
	if len(myHardRes) > 0 {
		retOptions["hard"] = myHardRes
		delete(retOptions, "l")
	}
	if len(mySoftRes) > 0 {
		retOptions["soft"] = mySoftRes
		delete(retOptions, "l")
	}
	return
}
