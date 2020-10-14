package slurm

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	flag "github.com/juju/gnuflag"

	core "jarvice.io/jarvice-hpc/core"
)

// Slurm CLI commands
const (
	SBatchName  = "sbatch"
	SCancelName = "scancel"
	SQueueName  = "squeue"
)

func SBatch(args []string) error {

	cmdJobSpec, cmdFlags, perr := parseSBatchArgs(args)
	if perr != nil {
		log.Fatal(perr)
		return perr
	}
	// jobscript required
	// sbatch [OPTIONS(0)...] [ : [OPTIONS(N)...]] script(0) [args(0)...]
	if cmdFlags.NArg() == 0 {
		return errors.New("sbatch: missing job script")
	}

	jobScriptFilename := cmdFlags.Args()[0]
	jobScript, jerr := core.ParseJobScript("SBATCH", jobScriptFilename)
	if jerr != nil {
		return errors.New("sbatch: WARNING unable to parse job script")
	}
	jobScriptFilename = filepath.Base(jobScriptFilename)
	// Save job script command line arguments
	jobScriptArgs := cmdFlags.Args()[1:]
	// TODO: remove
	fmt.Println(jobScriptArgs)
	// XXX
	// Process Slurm args inside job script
	scriptJobSpec, scriptFlags, perr := parseSBatchArgs(jobScript.Args)
	if perr != nil {
		log.Fatal(perr)
		return perr
	}
	// Go through set flags
	jobSpec := make(map[string]interface{})
	scriptFlags.Visit(func(f *flag.Flag) {
		key, err := lookupGnuArg(f.Name, scriptJobSpec)
		if err != nil {
			return
		}
		jobSpec[key] = f.Value.(flag.Getter).Get()
	})
	// Command line options override job script
	cmdFlags.Visit(func(f *flag.Flag) {
		key, err := lookupGnuArg(f.Name, cmdJobSpec)
		if err != nil {
			return
		}
		jobSpec[key] = f.Value.(flag.Getter).Get()
	})
	// Prompt user with unsupported options
	var sBatchUnsupported []string
	for k, _ := range jobSpec {
		if _, ok := sBatchSupportedArgs()[k]; !ok {
			sBatchUnsupported = append(sBatchUnsupported, k)
			delete(jobSpec, k)
		}
	}
	if len(sBatchUnsupported) > 0 {
		log.Printf("WARNING: %d unsupported options: %s", len(sBatchUnsupported), strings.Join(sBatchUnsupported, " "))
	}
	// Set Slurm Output Environment Variables
	if slurmSubmitDir, err := os.Getwd(); err != nil {
		log.Println("sbatch: WARNING setting SLURM_SUBMIT_DIR to ${HOME}")
		jobSpec["SLURM_SUBMIT_DIR"] = "${HOME}"
	} else {
		if value, ok := jobSpec["chdir"]; ok {
			jobSpec["SLURM_SUBMIT_DIR"] = value
			delete(jobSpec, "chdir")
		} else {
			jobSpec["SLURM_SUBMIT_DIR"] = slurmSubmitDir
		}
	}
	if submitHost, err := os.Hostname(); err != nil {
		log.Println("sbatch: WARNING setting SLURM_SUBMIT_HOST to localhost")
		jobSpec["SLURM_SUBMIT_HOST"] = "localhost"
	} else {
		jobSpec["SLURM_SUBMIT_HOST"] = submitHost
	}

	/*
		jobSpec := core.JobSpec{
			SubmitDirectory:   scriptJobSpec["SubmitDirectory"].Value.(string),
			SubmitHost:        scriptJobSpec["SubmitHost"].Value.(string),
			Queue:             *scriptJobSpec["partition"].Value.(*string),
			NodeCount:         *scriptJobSpec["nodes"].Value.(*int),
			CpuCount:          *scriptJobSpec["ntasks"].Value.(*int),
			WallClockLimit:    *scriptJobSpec["time"].Value.(*string),
			OutputFile:        *scriptJobSpec["output"].Value.(*string),
			ErrorFile:         *scriptJobSpec["error"].Value.(*string),
			CopyEnvironment:   *scriptJobSpec["export"].Value.(*string),
			EventNotification: *scriptJobSpec["mail-type"].Value.(*string),
			EmailAddress:      *scriptJobSpec["mail-user"].Value.(*string),
			JobName:           *scriptJobSpec["job-name"].Value.(*string),
			JobRestart:        *scriptJobSpec["requeue"].Value.(*bool),
			WorkingDirectory:  *scriptJobSpec["chdir"].Value.(*string),
			Exclusive:         *scriptJobSpec["exclusive"].Value.(*bool),
			Memory:            *scriptJobSpec["mem"].Value.(*string),
			ChargeAccount:     *scriptJobSpec["account"].Value.(*string),
			TasksPerNode:      *scriptJobSpec["tasks-per-node"].Value.(*int),
			CpusPerTask:       *scriptJobSpec["cpus-per-task"].Value.(*int),
			JobDependency:     *scriptJobSpec["dependency"].Value.(*string),
			JobProject:        *scriptJobSpec["wckey"].Value.(*string),
			GenericResources:  *scriptJobSpec["gres"].Value.(*string),
			Licenses:          *scriptJobSpec["licenses"].Value.(*string),
			BeginTime:         *scriptJobSpec["begin"].Value.(*string),
		}
	*/
	// TODO: remove
	fmt.Printf("\n\n%+v\n\n", map[string]interface{}{"jarvice_hpc": jobSpec})
	// XXX
	// Read JARVICE config for selected cluster
	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("sbatch: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	// TODO: remove fake queue
	myQueue := core.MyTestQueue()
	// XXX
	var clusterName string
	if val, ok := jobSpec["partition"]; ok {
		clusterName = val.(string)
	} else {
		clusterName = "default"
	}
	var cluster core.JarviceCluster
	if val, ok := myConfig[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("sbatch: cannot find credentials for " + clusterName)
	}
	userCreds := cluster.Creds
	// Use JARVICE UPLOAD parameter to transfer job script
	// XXX AppDef command must include 'command' and 'testUPLOAD' parameters
	// TODO: use dedicated JARVICE support to submit HPC jobs
	myParams := make(map[string]interface{})
	myParams["command"] = jobScript.Shell + " /opt/JARVICE/jobscript"
	myParams["testUPLOAD"] = base64.StdEncoding.EncodeToString(jobScript.Script)
	// XXX
	// Setup HPC job submission
	myApplication := core.JarviceApplication{
		Command:    myQueue.AppCommand,
		Geometry:   core.JarviceHpcGeometry,
		Parameters: myParams,
	}
	// Slurm --time option
	if val, ok := jobSpec["time"]; ok {
		myApplication.Walltime = val.(string)
	}

	myMachine := core.JarviceMachine{
		Type:  myQueue.DefaultMachine,
		Nodes: myQueue.DefaultMachineScale,
	}
	if val, ok := jobSpec["nodes"]; ok {
		myMachine.Nodes = val.(int)
	}
	// TODO: set ReadOnly and Force options?
	myVault := core.JarviceVault{
		Name:     cluster.Vault,
		ReadOnly: false,
		Force:    false,
	}

	myReq := core.JarviceJobRequest{
		App:         myQueue.App,
		Staging:     core.JarviceHpcStaging,
		Checkedout:  core.JarviceHpcCheckedout,
		Application: myApplication,
		Machine:     myMachine,
		Vault:       myVault,
		JobLabel:    jobScriptFilename,
		User:        userCreds,
	}
	// Slurm --job-name
	if val, ok := jobSpec["job-name"]; ok {
		myReq.JobLabel = val.(string)
	}
	// TODO: remove
	fmt.Printf("%+v\n", cluster.Endpoint)
	fmt.Printf("%+v\n", myReq)
	// XXX
	// Submit job request to JARVICE API
	var myJobResponse core.JarviceJobResponse
	if jobResponse, err := core.JarviceSubmitJob(cluster.Endpoint, myReq); err != nil {
		return errors.New("sbatch: " + err.Error())
	} else {
		myJobResponse = jobResponse
	}
	// TODO: use Slurm format
	fmt.Printf("Submitted job: %v\n", myJobResponse.Number)
	// XXX
	return nil
}

func SCancel(args []string) error {

	options, flags, perr := parseSCancelArgs(args)
	if perr != nil {
		log.Fatal(perr)
		return perr
	}

	if flags.NArg() == 0 {
		return errors.New("scancel: need to specify job ID")
	}
	// TODO: support multiple jobs
	if jobList := flags.Args()[1:]; len(jobList) > 0 {
		fmt.Println(jobList)
	}
	// XXX

	jobNumber := flags.Args()[0]

	// Go through set flags
	jobSpec := make(map[string]interface{})
	flags.Visit(func(f *flag.Flag) {
		key, err := lookupGnuArg(f.Name, options)
		if err != nil {
			return
		}
		jobSpec[key] = f.Value.(flag.Getter).Get()
	})

	var sCancelUnsupported []string
	for k, _ := range jobSpec {
		if _, ok := sCancelSupportedArgs()[k]; !ok {
			sCancelUnsupported = append(sCancelUnsupported, k)
			delete(jobSpec, k)
		}
	}
	if len(sCancelUnsupported) > 0 {
		log.Printf("WARNING: %d unsupported options: %s", len(sCancelUnsupported), strings.Join(sCancelUnsupported, " "))
	}

	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("scancel: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	var clusterName string
	if value, ok := jobSpec["partition"]; ok {
		clusterName = value.(string)
	} else {
		clusterName = "default"
	}
	cluster := myConfig[clusterName]
	// XXX
	userCreds := cluster.Creds
	// TODO: support multiple requests
	resp, err := http.Get(cluster.Endpoint + "/jarvice/terminate" +
		"?username=" + userCreds.Username + "&apikey=" + userCreds.Apikey +
		"&number=" + jobNumber)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New("scancel: unable to delete job")
	} else {
		fmt.Printf("Canceled job: %v\n", jobNumber)
	}
	return nil
}

func SQueue(args []string) error {
	// TODO: add slurm options
	flags := flag.NewFlagSet("squeue", flag.ContinueOnError)

	user := flags.String("u", "", "Slurm user")

	if flags.Parse(false, args) != nil {
		return errors.New("squeue: cannot process flags")
	}

	if flags.NArg() != 1 && len(*user) == 0 {
		return errors.New("squeue: invalid arguments")
	}

	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("squeue: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	// TODO: set cluster with --partition
	cluster := myConfig["default"]
	// XXX
	userCreds := cluster.Creds

	var jobNumber string
	if flags.NArg() > 0 {
		if number := flags.Args()[0]; len(number) > 0 {
			jobNumber = number
		}
	} else if *user != userCreds.Username {
		return errors.New("squeue: Slrum user does not match JARVICE credentials")
	}

	reqUrl := cluster.Endpoint + "/jarvice/status" +
		"?username=" + userCreds.Username + "&apikey=" + userCreds.Apikey

	if len(jobNumber) > 0 {
		reqUrl += "&number=" + jobNumber
	}

	resp, err := http.Get(reqUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New("squeue: unable to query job status")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("squeue: unable to query job status")
	}
	// TODO: use Slurm format
	fmt.Printf("\n%+v\n", string(body))
	// XXX
	return nil
}

// Slurm support Short and Long command line options
// Register both with the same Golang flag
func setFlagString(flags *flag.FlagSet, short, long, value, usage string) *string {
	flagVar := flags.String(short, value, usage)
	flags.StringVar(flagVar, long, value, usage)
	return flagVar
}

func setFlagInt(flags *flag.FlagSet, short, long string, value int, usage string) *int {
	flagVar := flags.Int(short, value, usage)
	flags.IntVar(flagVar, long, value, usage)
	return flagVar
}

func setFlagBool(flags *flag.FlagSet, short, long string, value bool, usage string) *bool {
	flagVar := flags.Bool(short, value, usage)
	flags.BoolVar(flagVar, long, value, usage)
	return flagVar
}
