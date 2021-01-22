package main

import (
	"errors"
	"fmt"

	"github.com/jessevdk/go-flags"
	jarvice "jarvice.io/core"
)

type JarviceConfigFlags struct {
	Help    bool   `short:"h" long:"help" description:"Show this help message"`
	Cluster string `short:"c" long:"cluster" description:"cluster name" default:"default"`
}

type JarviceCommand struct {
	Config  JarviceConfigFlags    `group:"Configuration Options"`
	Login   JarviceLoginCommand   `command:"login"`
	Vault   JarviceVaultCommand   `command:"vault"`
	Cluster JarviceClusterCommand `command:"cluster"`
}

type JarviceLoginCommand struct {
	Config   JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	Username string             `short:"u" long:"username" description:"JARVICE username"`
	Apikey   string             `short:"k" long:"apikey" description:"JARVICE apikey"`
	Vault    string             `short:"v" long:"vault" description:"JARVICE vault" default:"ephemeral"`
	Args     struct {
		Endpoint string `postitional-arg-name:"endpoint" description:"JARVICE API endpoint"`
	} `positional-args:"true" required:"1"`
}

type JarviceVaultCommand struct {
	Config JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	Vault  string             `short:"v" long:"vault" description:"JARVICE vault" default:"ephemeral"`
}

type JarviceClusterCommand struct {
	Config JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	List   bool               `short:"l" long:"list" description:"list available JARVICE configurations"`
}

var jarviceCommand JarviceCommand

func createHelpErr() error {
	err := flags.Error{
		Type:    flags.ErrHelp,
		Message: "show help message",
	}
	return &err
}

func (x *JarviceCommand) Execute(args []string) error {
	if x.Config.Help {
		return createHelpErr()
	}
	return nil
}

func (x *JarviceLoginCommand) Execute(args []string) error {
	if x.Config.Help {
		return createHelpErr()
	}
	return jarvice.HpcLogin(x.Args.Endpoint, x.Username, x.Apikey,
		x.Config.Cluster, x.Vault)
}

func (x *JarviceVaultCommand) Execute(args []string) error {
	if x.Config.Help {
		return createHelpErr()
	}
	return jarvice.HpcVault(x.Config.Cluster, x.Vault)
}

func (x *JarviceClusterCommand) Execute(args []string) error {
	if x.Config.Help {
		return createHelpErr()
	}
	config, _ := jarvice.ReadJarviceConfig()
	if x.List {
		for key, _ := range config {
			fmt.Println(key)
		}
		return nil
	}
	if _, ok := config[x.Config.Cluster]; ok {
		return jarvice.WriteJarviceConfigTarget(x.Config.Cluster)
	} else {
		return errors.New(x.Config.Cluster + " configuration does not exits")
	}
}

func init() {
	parser.AddCommand("jarvice",
		"JARVICE configuration",
		"The jarvice creates the configuration file to use with the JARVICE API",
		&jarviceCommand)
}
