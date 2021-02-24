package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
	jarvice "jarvice.io/core"
)

type SBatchCommand struct {
	Help      bool   `short:"h" long:"help" description:"Show this help message"`
	Chdir     string `short:"D" long:"chdir" description:"working directory"`
	Jobname   string `short:"J" long:"job-name" description:"Specify a name for the job allocation"`
	Nodes     int    `short:"N" long:"nodes" description:"Number of nodes be allocated to this job"`
	Time      string `short:"t" long:"time" description:"time limit hours:minutes:seconds"`
	Partition string `short:"p" long:"partition" description:"Request a specific partition for the resource allocation" default:"default"`
	Account   string `short:"A" long:"account" description:"Charge resources used by this job to specified account"`
	NodeInfo  string `short:"B" long:"extra-node-info" description:"Restrict node selection to nodes with at least the specified number of sockets, cores per socket and/or threads per core\nsockets[:cores[:threads]]\nNOTE: JARVICE does not accept socket or thread requests; cores request := sockets x cores"`
	Gpus      string `short:"G" long:"gpus" description:"Specify the total number of GPUs required for the job"`
	Mem       string `long:"mem" description:"Specify the real memory required per node. Default units are megabytes. Different units can be specified using the suffix [K|M|G|T]"`
	Gres      string `long:"gres" description:"Specifies a comma delimited list of generic consumable resources. The format of each entry on the list is \"name[[:type]:count]\""`
	Args      struct {
		JobScript []string `positional-arg-name:"jobscript" description:"job script | job command"`
		//JobCommand string `positional-arg-name:"command" description:
	} `positional-args:"true"`
}

var sBatchCommand SBatchCommand
var jobScriptParser = flags.NewNamedParser(jarvice.JobScriptArg,
	flags.PassDoubleDash|flags.IgnoreUnknown)

var jobScriptParserCommand SBatchCommand

type slurmGres struct {
	Type  string
	Count string
}

type slurmResources map[string]slurmGres

func parseSlurmResources(resources string) slurmResources {
	res := slurmResources{}

	for _, resource := range strings.Split(resources, ",") {
		split := strings.Split(resource, ":")
		if len(split) == 1 {
			res[split[0]] = slurmGres{
				Type: "true",
			}
		} else if len(split) == 2 {
			res[split[0]] = slurmGres{
				Type: split[1],
			}
		} else if len(split) == 3 {
			res[split[0]] = slurmGres{
				Type:  split[1],
				Count: split[2],
			}
		}

	}

	return res
}

func decodeMemReq(req string) (mem int, err error) {
	re := regexp.MustCompile("^[0-9]+")
	te := regexp.MustCompile("[KMGT]$")
	if match := re.FindString(req); len(match) > 0 {
		if base, perr := strconv.ParseInt(match, 10, 64); perr == nil {
			if mag := te.FindString(req); len(mag) > 0 {
				switch mag {
				case "K":
					mem = int(base) * 1024
				case "M":
					mem = int(base) * 1024 * 1024
				case "G":
					mem = int(base) * 1024 * 1024 * 1024
				case "T":
					mem = int(base) * 1024 * 1024 * 1024 * 1024
				}

			} else {
				mem = int(base) * 1024 * 1024
			}
			mem = int(math.Ceil(float64(mem) / float64((1024 * 1024 * 1024))))
			return
		}
	}
	err = errors.New("Invalid mem request")
	return
}

func (x *SBatchCommand) Execute(args []string) error {
	// leave early if parsing jobscript arguments
	if jobScriptParser.Active != nil &&
		jobScriptParser.Active.Name == jarvice.JobScriptArg {
		return nil
	}

	if x.Help {
		return jarvice.CreateHelpErr()
	}

	// Set jobscript name
	jobScriptFilename := "STDIN"
	submitCommand := "STDIN"
	if len(x.Args.JobScript) == 1 {
		jobScriptFilename = x.Args.JobScript[0]
	} else if len(x.Args.JobScript) > 1 {
		submitCommand = strings.Join(x.Args.JobScript, " ")
	}

	var jobScript jarvice.JobScript

	if len(jobScriptFilename) > 0 {
		if val, jerr := jarvice.ParseJobScript("SBATCH", jobScriptFilename); jerr != nil {
			return errors.New("sbatch: WARNING unable to parse job script")
		} else {
			jobScript = val
		}
	} else {
		jobScript = jarvice.JobScript{
			Shell:  "/bin/sh",
			Script: []byte(submitCommand),
		}
	}
	// parse flags from jobscript (CLI flags take precedence;override == false)
	if jarvice.ParseJobFlags(x,
		parser,
		jobScriptParser,
		append([]string{jarvice.JobScriptArg}, jobScript.Args...),
		false) != nil {
		// Best effort
		fmt.Println("WARNING: unable to parse flags in jobscript")
	}

	jobScriptFilename = filepath.Base(jobScriptFilename)

	resources := parseSlurmResources(x.Gres)

	// Read JARVICE config for selected cluster
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return nil
	}

	queueName := x.Partition
	// need JARVICE API creds, 'info', and 'name' for /jarvice/queues request
	urlValues := cluster.GetUrlCreds()
	urlValues.Add("info", "true")
	urlValues.Add("name", queueName)
	jarviceQueues := make(map[string]jarvice.JarviceQueue)
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"queues",
		urlValues); err == nil {

		if err := json.Unmarshal(resp, &jarviceQueues); err != nil {
			return errors.New("sbatch: " + err.Error())
		}
	} else {
		return errors.New("sbatch: connot find partition: " + queueName + "  " + err.Error())
	}
	var myQueue jarvice.JarviceQueue
	for _, queue := range jarviceQueues {
		myQueue = queue
		break
	}

	var cwd string
	if len(x.Chdir) > 0 {
		cwd = x.Chdir
	}
	jobName := "SBATCH"
	if len(x.Jobname) > 0 {
		jobName = x.Jobname
	}

	// Set /etc/hosts for remote HPC job
	// Best effort
	hostnameCmd, _ := exec.Command("hostname").Output()
	myHostname := strings.TrimSuffix(string(hostnameCmd), "\n")
	myPrivateIP := strings.TrimSuffix(jarvice.GetOutboundIP(), "\n")
	var ipString string
	if len(myPrivateIP) > 0 {
		ipString = myPrivateIP + " " + myHostname
	}
	// Set Slurm Output Environment Variables
	slurmEnvs := make(map[string]string)
	slurmEnvs["SLURM_CLUSTER_NAME"] = jarvice.ReadJarviceConfigTarget()
	if len(x.Account) > 0 {
		slurmEnvs["SLURM_JOB_ACCOUNT"] = x.Account
	} else {
		slurmEnvs["SLURM_JOB_ACCOUNT"] = cluster.Creds.Username
	}
	if len(x.Partition) > 0 {
		slurmEnvs["SLURM_JOB_PARTITION"] = x.Partition
	} else {
		slurmEnvs["SLURM_JOB_PARTITION"] = "default"
	}
	if slurmSubmitDir, err := os.Getwd(); err != nil {
		log.Println("sbatch: WARNING setting SLURM_SUBMIT_DIR to ${HOME}")
		slurmEnvs["SLURM_SUBMIT_DIR"] = "${HOME}"
	} else {
		if len(x.Chdir) > 0 {
			slurmEnvs["SLURM_SUBMIT_DIR"] = x.Chdir
		} else {
			slurmEnvs["SLURM_SUBMIT_DIR"] = slurmSubmitDir
		}
	}
	if submitHost, err := os.Hostname(); err != nil {
		log.Println("sbatch: WARNING setting SLURM_SUBMIT_HOST to localhost")
		slurmEnvs["SLURM_SUBMIT_HOST"] = "localhost"
	} else {
		slurmEnvs["SLURM_SUBMIT_HOST"] = submitHost
	}
	if len(x.Jobname) > 0 {
		slurmEnvs["SLURM_JOB_NAME"] = x.Jobname
	} else {
		slurmEnvs["SLURM_JOB_NAME"] = jobScriptFilename
	}
	myHpcReq := jarvice.HpcReq{
		// sudo is required to edit /etc/hosts (best effort)
		JobEnvConfig: `join () { local IFS="$1"; shift; echo "$*"; };` +
			`ips=$(cat /var/JARVICE/c/hosts | awk '{print $1}' | xargs);` +
			`hosts=$(cat /var/JARVICE/c/hosts | awk '{print $2}' | xargs);` +
			`sge_hosts="$(join , $hosts)";` +
			`numcpu="$(cat /etc/JARVICE/cores | grep $(hostname) | wc -l)";` +
			`numnodes="$(cat /etc/JARVICE/nodes | wc -l )";` +
			`cpupernode="$(( $(cat /etc/JARVICE/cores | wc -l) / $(cat /etc/JARVICE/nodes | wc -l) ))";` +
			`procid="$(ps axo pid,command | grep '/bin/sh -l -c join ()' | awk 'NR==1{print $1}')";` +
			`echo ` + ipString + ` | sudo tee -a /etc/hosts || true`,
		JobScript: base64.StdEncoding.EncodeToString(jobScript.Script),
		JobShell: "cd " + cwd + " && " +
			"SLURM_JOB_NODELIST=${slurm_hosts} " +
			"SLURM_NODELIST=${slurm_hosts} " +
			"SLURM_NODE_ALIASES=${host_alias} " +
			"SLURMD_NODENAME=${slurm_host} " +
			"SLURM_CPUS_ON_NODE=${numcpu} " +
			"SLURM_JOB_NUM_NODES=${numnodes} " +
			"SLURM_NNODES=${numnodes} " +
			"SLURM_JOB_CPUS_PER_NODE=${cpupernode} " +
			"SLURM_PROCID=${procid} " +
			jobScript.Shell,
		Queue:     myQueue.Name,
		Umask:     0,
		Envs:      slurmEnvs,
		Resources: map[string]string{},
	}

	// Parse resource request
	// Check for machine type
	var hpcMachineReq string
	if val, ok := resources["mc_name"]; ok {
		hpcMachineReq = val.Type
	}
	myHpcReq.Resources["mc_name"] = hpcMachineReq
	// Check for licenses
	var hpcLicenses *string
	if val, ok := resources["mc_licenses"]; ok {
		hpcLicenses = new(string)
		*hpcLicenses = val.Type
	}
	// Check for account (i.e. project)
	var jobProject *string
	if len(x.Account) > 0 {
		jobProject = new(string)
		*jobProject = x.Account
	}

	// CPU cores
	coreReq := 0
	if val := x.NodeInfo; len(val) > 0 {
		// Grab first value as cores request and discard the rest
		cores := strings.Split(val, ":")
		req := []float64{0.0, 1.0}
		for index, number := range cores {
			if index > 2 {
				break
			}
			if f, err := strconv.ParseFloat(number, 64); err == nil {
				req[index] = f
			} else {
				req[index] = 0.0
			}
		}
		coreReq = int(math.Ceil(req[0] * req[1]))
	}
	myHpcReq.Resources["mc_cores"] = strconv.FormatInt(int64(coreReq), 10)
	// RAM
	memReq := 0
	if val := x.Mem; len(val) > 0 {
		if i, err := decodeMemReq(val); err == nil {
			memReq = i
		}
	}
	myHpcReq.Resources["mc_ram"] = strconv.FormatInt(int64(memReq), 10)
	// Setup HPC job submission
	myApplication := jarvice.JarviceApplication{
		Command:  jarvice.JarviceHpcCommandName,
		Geometry: jarvice.JarviceHpcGeometry,
	}
	// need to validate scale (positive integer)
	nodeScale := 1
	if x.Nodes > 0 {
		nodeScale = x.Nodes
	}
	// check if scale request is larger than queue size
	if nodeScale > myQueue.MachineScale {
		return errors.New("sbatch: -Nodes request larger than partition size (" +
			strconv.Itoa(myQueue.MachineScale) + ")")
	}
	myMachine := jarvice.JarviceMachine{
		Type:  myQueue.DefaultMachine,
		Nodes: nodeScale,
	}
	// TODO: set ReadOnly and Force options?
	myVault := jarvice.JarviceVault{
		Name:     cluster.Vault,
		ReadOnly: false,
		Force:    false,
	}

	userCreds := cluster.Creds

	myReq := jarvice.JarviceJobRequest{
		App:         myQueue.App,
		Staging:     jarvice.JarviceHpcStaging,
		Checkedout:  jarvice.JarviceHpcCheckedout,
		Application: myApplication,
		Machine:     myMachine,
		Vault:       myVault,
		JobLabel:    jobName,
		User:        userCreds,
		Hpc:         myHpcReq,
		Licenses:    hpcLicenses,
		JobProject:  jobProject,
	}
	// SgeJobReqDebug(myReq)
	// Submit job request to JARVICE API
	var myJobResponse jarvice.JarviceJobResponse
	if jobResponse, err := jarvice.JarviceSubmitJob(cluster.Endpoint, myReq); err != nil {
		return errors.New("sbatch: " + err.Error())
	} else {
		myJobResponse = jobResponse
	}
	fmt.Printf("Your job %d (\"%s\") has been submitted\n", int(myJobResponse.Number), jobScriptFilename)

	return nil

}

func init() {
	parser.AddCommand("sbatch",
		"Slurm sbatch",
		"Submit a batch script to Slurm",
		&sBatchCommand)
	// parser for jobscript flags
	jobScriptParser.AddCommand(jarvice.JobScriptArg,
		jarvice.JobScriptArg,
		jarvice.JobScriptArg,
		&jobScriptParserCommand)
}
