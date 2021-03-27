package main

import (
	"errors"
	"fmt"

	jarvice "jarvice.io/core"
)

type JarviceConfigFlags struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

type JarviceCommand struct {
	Config  JarviceConfigFlags    `group:"Configuration Options"`
	Login   JarviceLoginCommand   `command:"login"`
	Vault   JarviceVaultCommand   `command:"vault"`
	Cluster JarviceClusterCommand `command:"cluster"`
}

type JarviceLoginCommand struct {
	Config   JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	Vault    string             `short:"v" long:"vault" description:"JARVICE vault" default:"ephemeral"`
	Insecure bool               `short:"k" long:"insecure" description:"proceed if server configuration is considered insecure"`
	Args     struct {
		Endpoint string `postitional-arg-name:"endpoint" description:"JARVICE API endpoint"`
		Cluster  string `positional-arg-name:"cluster" description:"JARVICE cluster"`
		Username string `positional-arg-name:"username "description:"JARVICE username"`
		Apikey   string `positinal-arg-name:"apikey" description:"JARVICE apikey"`
	} `positional-args:"true" required:"4"`
}

type JarviceVaultCommand struct {
	Config JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	Vault  string             `short:"v" long:"vault" description:"JARVICE vault" default:"ephemeral"`
}

type JarviceClusterCommand struct {
	Config JarviceConfigFlags `group:"Configuration Options" hidden:"true"`
	List   bool               `short:"l" long:"list" description:"list available JARVICE configurations"`
	Args   struct {
		Cluster string `positional-arg-name:"cluster" description:"JARVICE cluster"`
	} `positional-args:"true"`
}

var jarviceCommand JarviceCommand

func (x *JarviceCommand) Execute(args []string) error {
	if x.Config.Help {
		return jarvice.CreateHelpErr()
	}
	return nil
}

func (x *JarviceLoginCommand) Execute(args []string) error {
	if x.Config.Help {
		return jarvice.CreateHelpErr()
	}
	return jarvice.HpcLogin(x.Args.Endpoint, x.Insecure, x.Args.Cluster,
		x.Args.Username, x.Args.Apikey, x.Vault)
}

func (x *JarviceVaultCommand) Execute(args []string) error {
	if x.Config.Help {
		return jarvice.CreateHelpErr()
	}
	return jarvice.HpcVault(x.Vault)
}

func (x *JarviceClusterCommand) Execute(args []string) error {
	if x.Config.Help {
		return jarvice.CreateHelpErr()
	}
	config, _ := jarvice.ReadJarviceConfig()
	if x.List {
		if len(config) == 0 {
			return errors.New("No clusters found. Setup config using: jarvice login")
		}
		for key, _ := range config {
			fmt.Println(key)
		}
		return nil
	}
	if len(x.Args.Cluster) == 0 {
		return errors.New("Cluster argument missing")
	}
	if _, ok := config[x.Args.Cluster]; ok {
		return jarvice.WriteJarviceConfigTarget(x.Args.Cluster)
	} else {
		return errors.New(x.Args.Cluster + " configuration does not exits." +
			" Setup config using: jarvice login")
	}
}

func init() {
	parser.AddCommand("jarvice",
		"JARVICE configuration",
		"The jarvice creates the configuration file to use with the JARVICE API",
		&jarviceCommand)
}
