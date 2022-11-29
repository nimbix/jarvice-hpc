package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
	jarvice "jarvice.io/jarvice-hpc/core"
)

type QSubCommand struct {
	Help      bool     `short:"h" long:"help" description:"Show this help message"`
	Binary    string   `short:"b" description:"binary or script"`
	Shell     string   `short:"S" description:"job shell"`
	JobName   string   `short:"N" description:"job name"`
	Cwd       bool     `long:"cwd" description:"current working directory"`
	Resources []string `short:"l" description:"job resources. NOTE: -soft treated as hard resources"`
	Pe        int      `long:"pe" description:"parallel environment job scale.\n-pe <pe-name> <pe-scale>\nNOTE: ranges not support (expect single integer)\n<pe-name> will be discarded"`
	Queue     string   `short:"q" description:"target queue" default:"default"`
	Project   string   `short:"P" description:"Specifies the project to which this  job  is  assigned."`
	Output    string   `short:"o" description:"Output file."`
	Error     string   `short:"e" description:"Error file."`
	Args      struct {
		JobScript []string `positional-arg-name:"jobscript" description:"SGE job script | job command"`
		//JobCommand string `positional-arg-name:"command" description:
	} `positional-args:"true"`
}

var qSubCommand QSubCommand
var jobScriptParser = flags.NewNamedParser(jarvice.JobScriptArg,
	flags.PassDoubleDash|flags.IgnoreUnknown)

var jobScriptParserCommand QSubCommand

func splitAtCommas(s string) []string {
	res := []string{}
	var beg int
	var inString bool

	for i := 0; i < len(s); i++ {
		if s[i] == ',' && !inString {
			res = append(res, s[beg:i])
			beg = i + 1
		} else if s[i] == '"' {
			if !inString {
				inString = true
			} else if i > 0 && s[i-1] != '\\' {
				inString = false
			}
		}
	}
	return append(res, s[beg:])
}

func parseSgeResources(resources []string) map[string]string {
	res := map[string]string{}
	for _, resource := range resources {
		for _, flag := range splitAtCommas(resource) {
			split := strings.Split(flag, "=")
			// save valid pairs (foo=bar)
			if len(split) == 2 {
				res[split[0]] = strings.Trim(split[1], "\"")
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

func (x *QSubCommand) Execute(args []string) error {
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
		jobScriptFilename = ""
	}

	// validate binary flag
	if jarvice.IsYes(x.Binary) && jobScriptFilename == "STDIN" {
		return &jarvice.SgeError {
			Command: "qsub",
			Err: errors.New("missing command"),
		}
	}

	var jobScript jarvice.JobScript

	if len(jobScriptFilename) > 0 {
		if val, jerr := jarvice.ParseJobScript("$", jobScriptFilename); jerr != nil {
			return &jarvice.SgeError {
				Command: "qsub",
				Err: errors.New("WARNING unable to parse job script"),
			}
		} else {
			jobScript = val
		}
	} else {
		jobScript = jarvice.JobScript{
			Shell:  "/bin/sh",
			Script: []byte(submitCommand),
		}
	}
	// workaround for slice copy
	if len(x.Resources) < 1 {
		sliceMax := len(jobScript.Args) + 10
		x.Resources = make([]string, sliceMax, sliceMax)
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

	resources := parseSgeResources(x.Resources)

	// Read JARVICE config for selected cluster
	cluster, err := jarvice.GetClusterConfig()
	if err != nil {
		return &jarvice.SgeError {
			Command: "qsub",
			Err: err,
		}
	}
	queueName := x.Queue
	// need JARVICE API creds, 'info', and 'name' for /jarvice/queues request
	urlValues := cluster.GetUrlCreds()
	urlValues.Add("info", "true")
	urlValues.Add("name", queueName)
	jarviceQueues := make(map[string]jarvice.JarviceQueue)
	if resp, err := jarvice.ApiReq(cluster.Endpoint,
		"queues",
		cluster.Insecure,
		urlValues); err == nil {
		if err := json.Unmarshal(resp, &jarviceQueues); err != nil {
			return &jarvice.SgeError {
				Command: "qsub",
				Err: errors.New("failed to read queues"),
			}
		}
	} else {
		return &jarvice.SgeError {
			Command: "qsub",
			Err: errors.New("cannot find queue: " + queueName + "  " + err.Error()),
		}
	}
	var myQueue jarvice.JarviceQueue
	for _, queue := range jarviceQueues {
		myQueue = queue
		break
	}

	if len(x.Shell) > 0 {
		jobScript.Shell = x.Shell
	}
	// Setup I/O redirect using block { ... }
	jobScript.Script = append([]byte{'{', '\n'}, jobScript.Script...)
	jobScript.Script = append(jobScript.Script, '\n', '}')
	if len(x.Output) > 0 {
		jobScript.Script = append(jobScript.Script, []byte(" >"+x.Output)...)
	}
	if len(x.Error) > 0 {
		jobScript.Script = append(jobScript.Script, []byte(" 2>"+x.Error)...)
	}

	var cwd string
	if x.Cwd {
		if wd, err := os.Getwd(); err != nil {
			cwd = "${HOME}"
		} else {
			cwd = wd
		}
	}
	jobName := "SGE"
	if len(x.JobName) > 0 {
		jobName = x.JobName
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
	// Set SGE Output Environment Variables
	sgeEnvs := make(map[string]string)
	blacklisted_envs := map[string]struct{}{
		"PATH":                 struct{}{},
		"USER":                 struct{}{},
		"HOME":                 struct{}{},
		"EDITOR":               struct{}{},
		"UID":                  struct{}{},
		"TERM":                 struct{}{},
		"SHELL":                struct{}{},
		"HOSTNAME":             struct{}{},
		"GLAD":                 struct{}{},
		"JARVICE_HEALTH_PORT":  struct{}{},
		"JARVICE_HPC_LOGLEVEL": struct{}{},
		"JARVICE_ID_GID":       struct{}{},
		"JARVICE_ID_GROUP":     struct{}{},
		"JARVICE_ID_UID":       struct{}{},
		"JARVICE_ID_USER":      struct{}{},
		"JARVICE_INGRESSPATH":  struct{}{},
		"JARVICE_JOBTOKEN":     struct{}{},
		"JARVICE_MPI_CMA":      struct{}{},
		"JARVICE_MPI_PROVIDER": struct{}{},
		"JARVICE_TOOLS":        struct{}{},
		"JARVICE_TOOLS_BIN":    struct{}{},
		"JARVICE_VAULT_NAME":   struct{}{},
		"JOB_LABEL":            struct{}{},
		"JOB_NAME":             struct{}{},
		"JOB_PRIVATEIP":        struct{}{},
		"JOB_PUBLICIP":         struct{}{},
		"MODULEPATH":           struct{}{},
		"MODULESHOME":          struct{}{},
	}
	for _, env := range os.Environ() {
		parts := strings.Split(env, "=")
		if len(parts) == 2 {
			if _, ok := blacklisted_envs[parts[0]]; !ok &&
				!strings.Contains(parts[0], "BASH_FUNC") &&
				!strings.Contains(parts[0], "KUBERNETES_") {
				sgeEnvs[parts[0]] = parts[1]
			}
		}
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
			"SGE_JOB_NODELIST=${sge_hosts} " +
			"SGE_CPUS_ON_NODE=${numcpu} " +
			"SGE_JOB_NUM_NODES=${numnodes} " +
			"SGE_JOB_CPUS_PER_NODE=${cpupernode} " +
			"SGE_PROCID=${procid} " +
			jobScript.Shell,
		Queue:     myQueue.Name,
		Umask:     0,
		Envs:      sgeEnvs,
		Resources: map[string]string{},
	}

	// Parse resource request
	// Check for machine type
	var hpcMachineReq string
	if val, ok := resources["mc_name"]; ok {
		hpcMachineReq = val
	}
	myHpcReq.Resources["mc_name"] = hpcMachineReq
	// Check for licenses
	var hpcLicenses *string
	if val, ok := resources["mc_licenses"]; ok {
		hpcLicenses = new(string)
		*hpcLicenses = val
	}
	// Check for project
	var jobProject *string
	if val, ok := resources["mc_project"]; ok || len(x.Project) > 0 {
		jobProject = new(string)
		if ok {
			*jobProject = val
		}  else {
			*jobProject = x.Project
		}
	}

	// CPU cores
	coreReq := 0
	if val, ok := resources["cpu"]; ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			coreReq = int(math.Ceil(f))
		}
	}
	myHpcReq.Resources["mc_cores"] = strconv.FormatInt(int64(coreReq), 10)
	// RAM
	memReq := 0
	if val, ok := resources["h_rss"]; ok {
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
	if x.Pe > 0 {
		nodeScale = x.Pe
	}
	// check if -pe request is larger than queue size
	if nodeScale > myQueue.MachineScale {
		return &jarvice.SgeError {
			Command: "qsub",
			Err: errors.New("-pe request larger than queue size (" +
				strconv.Itoa(myQueue.MachineScale) + ")"),
			}
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
	// Submit job request to JARVICE API
	var myJobResponse jarvice.JarviceJobResponse
	if jobResponse, err := jarvice.JarviceSubmitJob(cluster.Endpoint,
		cluster.Insecure, myReq); err != nil {
		return &jarvice.SgeError {
			Command: "qsub",
			Err: err,
		}
	} else {
		myJobResponse = jobResponse
	}
	fmt.Printf("Your job %d (\"%s\") has been submitted\n", int(myJobResponse.Number), jobScriptFilename)

	return nil

}

func init() {
	parser.AddCommand("qsub",
		"SGE qsub",
		"submit a batch job to Sun Grid Engine",
		&qSubCommand)
	// parser for jobscript flags
	jobScriptParser.AddCommand(jarvice.JobScriptArg,
		jarvice.JobScriptArg,
		jarvice.JobScriptArg,
		&jobScriptParserCommand)
}
