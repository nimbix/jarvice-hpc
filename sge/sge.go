package sge

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	core "jarvice.io/jarvice-hpc/core"
)

// CLI commands
const (
	QStatName = "qstat"
	QConfName = "qconf"
	QSubName  = "qsub"
	QAcctName = "qacct"
)

func QSub(args []string) error {

	cmdJobSpec, jobScriptFilename, submitCommand, perr := parseQSubArgs(args)
	if perr != nil {
		log.Fatal(perr)
		return perr
	}
	var jobScript JobScript
	if val, ok := cmdJobSpec["b"]; ok {
		if val == "y" {
			// Job Script not on submission hosts
			// Get command with args
			if submitCommand == "STDIN" {
				return errors.New("qsub: missing command")
			}
		}
	}
	if len(jobScriptFilename) >= 0 {
		if val, jerr := core.ParseJobScript("$", jobScriptFilename); jerr != nil {
			return errors.New("qsub: WARNING unable to parse job script")
		} else {
			jobScript = val
		}
	} else {
		jobScript = core.JobScript{
			Shell:  "/bin/sh",
			Script: []byte(submitCommand),
		}
	}
	SgeJobScriptDebug(jobScript)
	jobScriptFilename = filepath.Base(jobScriptFilename)
	// Save job script command line arguments
	// jobScriptArgs := cmdFlags.Args()[1:]
	// Process Slurm args inside job script
	scriptJobSpec, _, _, perr := parseQSubArgs(jobScript.Args)
	if perr != nil {
		log.Fatal(perr)
		return perr
	}
	SgeJobSpecDebug(scriptJobSpec)
	// Go through set flags
	jobSpec := make(map[string]interface{})
	for k, v := range scriptJobSpec {
		jobSpec[k] = v
	}
	for k, v := range cmdJobSpec {
		jobSpec[k] = v
	}
	SgeJobCmdDebug(jobSpec)
	// Prompt user with unsupported options
	var qSubUnsupported []string
	for k, _ := range jobSpec {
		if _, ok := qSubSupportedArgs()[k]; !ok {
			qSubUnsupported = append(qSubUnsupported, k)
			delete(jobSpec, k)
		}
	}
	if len(qSubUnsupported) > 0 {
		// log.Printf("WARNING: %d unsupported options: %s", len(qSubUnsupported), strings.Join(qSubUnsupported, " "))
	}
	var hardDebug, softDebug arrayFlags
	var hardResources, softResources map[string]string
	if val, ok := jobSpec["hard"]; ok {
		hardResources = make(map[string]string)
		for _, flag := range val.(arrayFlags) {
			tmpHard := strings.Split(flag, ",")
			for _, entry := range tmpHard {
				parts := strings.Split(entry, "=")
				if len(parts) == 1 {
					hardResources[entry] = "true"
				} else if len(parts) == 2 {
					hardResources[parts[0]] = parts[1]
				}
			}
		}
		hardDebug = val.(arrayFlags)
	}
	if val, ok := jobSpec["soft"]; ok {
		softResources = make(map[string]string)
		for _, flag := range val.(arrayFlags) {
			tmpSoft := strings.Split(flag, ",")
			for _, entry := range tmpSoft {
				parts := strings.Split(entry, "=")
				if len(parts) == 1 {
					softResources[entry] = "true"
				} else if len(parts) == 2 {
					softResources[parts[0]] = parts[1]
				}
			}
		}
		softDebug = val.(arrayFlags)
	}
	SgeJobSpecDebug(jobSpec)
	SgeJobResDebug(hardDebug, softDebug)
	// Read JARVICE config for selected cluster
	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("qsub: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	var clusterName string
	if val, ok := hardResources["mc_cluster"]; ok {
		clusterName = val
	} else if val, ok := softResources["mc_cluster"]; ok {
		clusterName = val
	} else {
		clusterName = "default"
	}
	var queueName string
	if val, ok := jobSpec["q"]; ok {
		queueName = val.(string)
	} else {
		queueName = "default"
	}
	var cluster core.JarviceCluster
	if val, ok := myConfig[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("qsub: cannot find credentials for " + clusterName)
	}
	userCreds := cluster.Creds
	urlString := cluster.Endpoint + "/jarvice/queues?username=" + userCreds.Username + "&apikey=" + userCreds.Apikey + "&info=true&name=" + queueName
	res, err := http.Get(urlString)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errors.New("qsub: " + res.Status + " cannot find queue: " + queueName)
	} else if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		errResp := map[string]string{}
		var errMsg string
		if err := json.Unmarshal([]byte(body), &errResp); err == nil {
			if msg, ok := errResp["error"]; ok {
				errMsg = ": " + msg
			}
		}
		return errors.New("qsub: " + res.Status + errMsg)
	}

	jarviceQueues := make(map[string]core.JarviceQueue)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("qsub: " + err.Error())
	}
	if err := json.Unmarshal([]byte(body), &jarviceQueues); err != nil {
		return errors.New("qsub: " + err.Error())
	}
	if len(jarviceQueues) < 1 {
		return errors.New("qsub: cannot find queue: " + queueName)
	}
	var myQueue core.JarviceQueue
	for _, queue := range jarviceQueues {
		myQueue = queue
		break
	}
	if val, ok := jobSpec["S"]; ok {
		jobScript.Shell = val.(string)
	}
	var cwd string
	if _, ok := jobSpec["cwd"]; ok {
		if wd, err := os.Getwd(); err != nil {
			cwd = "${HOME}"
		} else {
			cwd = wd
		}
	}
	jobName := "SGE"
	if val, ok := jobSpec["N"]; ok {
		jobName = val.(string)
	}
	// Set /etc/hosts for remote HPC job
	// Best effort
	hostnameCmd, _ := exec.Command("hostname").Output()
	myHostname := strings.TrimSuffix(string(hostnameCmd), "\n")
	myPrivateIP := strings.TrimSuffix(core.GetOutboundIP(), "\n")
	var ipString string
	if len(myPrivateIP) > 0 {
		ipString = myPrivateIP + " " + myHostname
	}
	// Set Slurm Output Environment Variables
	sgeEnvs := make(map[string]string)
	myHpcReq := core.HpcReq{
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
			"SGE_JOB_NODELIST=${slurm_hosts} " +
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
	if val, ok := softResources["mc_name"]; ok {
		hpcMachineReq = val
	}
	if val, ok := hardResources["mc_name"]; ok {
		hpcMachineReq = val
	}
	myHpcReq.Resources["mc_name"] = hpcMachineReq
	// Check for licenses
	var hpcLicenses *string
	if val, ok := softResources["mc_licenses"]; ok {
		hpcLicenses = new(string)
		*hpcLicenses = val
	}
	if val, ok := hardResources["mc_licenses"]; ok {
		if hpcLicenses == nil {
			hpcLicenses = new(string)
		}
		*hpcLicenses = val
	}
	// CPU cores
	// TODO: fill in from request
	coreReq := 16
	if val, ok := hardResources["cpu"]; ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			coreReq = int(math.Ceil(f))
		}
	} else if val, ok := softResources["cpu"]; ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			coreReq = int(math.Ceil(f))
		}
	}
	myHpcReq.Resources["mc_cores"] = strconv.FormatInt(int64(coreReq), 10)
	// RAM
	memReq := 40
	if val, ok := hardResources["h_rss"]; ok {
		if i, err := decodeMemReq(val); err == nil {
			memReq = i
		}
	} else if val, ok := softResources["h_rss"]; ok {
		if i, err := decodeMemReq(val); err == nil {
			memReq = i
		}
	}
	myHpcReq.Resources["mc_ram"] = strconv.FormatInt(int64(memReq), 10)
	// XXX
	// Setup HPC job submission
	myApplication := core.JarviceApplication{
		Command:  core.JarviceHpcCommandName,
		Geometry: core.JarviceHpcGeometry,
	}
	nodeScale := 1
	if val, ok := jobSpec["pe"]; ok {
		splitScale := strings.Split(val.(string), "=")
		if len(splitScale) > 1 {
			if i, err := strconv.Atoi(splitScale[1]); err == nil {
				nodeScale = i
			}
		}
	}
	myMachine := core.JarviceMachine{
		Type:  myQueue.DefaultMachine,
		Nodes: nodeScale,
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
		JobLabel:    jobName,
		User:        userCreds,
		Hpc:         myHpcReq,
		Licenses:    hpcLicenses,
	}
	SgeJobReqDebug(myReq)
	// Submit job request to JARVICE API
	var myJobResponse core.JarviceJobResponse
	if jobResponse, err := core.JarviceSubmitJob(cluster.Endpoint, myReq); err != nil {
		return errors.New("qsub: " + err.Error())
	} else {
		myJobResponse = jobResponse
	}
	fmt.Printf("Your job %d (\"%s\") has been submitted\n", int(myJobResponse.Number), jobScriptFilename)
	return nil
}

// Debug
func SgeCliDebug() {
	f, err := os.OpenFile("/tmp/sge.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	// best effort
	if err == nil {
		defer f.Close()
		dt := time.Now()
		f.WriteString(dt.String() + " " + strings.Join(os.Args, " ") + "\n")
	}
	return
}

func SgeJobScriptDebug(job core.JobScript) {
	f, err := os.OpenFile("/tmp/sgeJob.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("%+v\n", job))
		f.WriteString(string(job.Script))
	}
	return
}

func SgeJobReqDebug(req core.JarviceJobRequest) {
	f, err := os.OpenFile("/tmp/sgeReq.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("%+v\n", req))
	}
	b, _ := json.Marshal(req)
	ff, err2 := os.OpenFile("/tmp/sge.json", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err2 == nil {
		defer ff.Close()
		ff.WriteString(fmt.Sprintf("%s\n", string(b)))
	}
	return
}

func SgeJobCmdDebug(cmd map[string]interface{}) {
	f, err := os.OpenFile("/tmp/sgeCmd.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("%+v\n", cmd))
	}
	return
}

func SgeJobResDebug(hardRes, softRes []string) {
	f, err := os.OpenFile("/tmp/sgeRes.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer f.Close()
		if len(hardRes) > 0 {
			f.WriteString(fmt.Sprintf("hard resources: %+v\n", hardRes))
		}
		if len(softRes) > 0 {
			f.WriteString(fmt.Sprintf("soft resources: %+v\n", softRes))
		}
	}
}

func SgeJobSpecDebug(spec map[string]interface{}) {
	f, err := os.OpenFile("/tmp/sgeSpec.out", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("jobSpec: \n\n%+v\n\n", spec))
	}
}

func qstatVersion() {
	fmt.Println("SGE 8.1.9")
	return
}

func qstatUser(user string) error {
	// Read JARVICE config for selected cluster
	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("qsub: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	clusterName := "default"
	var cluster core.JarviceCluster
	if val, ok := myConfig[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("qstat: cannot find credentials for " + clusterName)
	}
	userCreds := cluster.Creds
	urlString := cluster.Endpoint + "/jarvice/jobs?username=" + userCreds.Username + "&apikey=" + userCreds.Apikey
	res, err := http.Get(urlString)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("qstat: cannot query queue")
	}
	var jarviceJobs core.JarviceJobs
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("qstat: cannot query queue")
	}
	if err := json.Unmarshal([]byte(body), &jarviceJobs); err != nil {
		return errors.New("qstat: cannot query queue")
	}
	retTable := [][]interface{}{
		{"job-ID", "prior", "name", "user", "state", "submit/start at", "queue"},
	}

	sgeState := "qw"
	for index, job := range jarviceJobs {
		switch job.Status {
		case "PROCESSING STARTING":
			sgeState = "r"
			/*
				case "SUBMITTED":
				case "COMPLETED":
				case "COMPLETED WITH ERROR":
				case "TERMINATED":
				case "CANCELED":
				case "EXEMPT":
				case "SEQUENTIALLY QUEUED":
			*/
		default:
			sgeState = "qw"
		}
		subTime := time.Unix(int64(job.SubmitTime), 0)
		retTable = append(retTable, []interface{}{index, "0", job.Label, job.User, sgeState, subTime.Format(time.UnixDate), job.ApiSubmission.Queue})
	}
	/*
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		w.Comma = '\t'
		for _, record := range retTable {
			if err := w.Write(record); err != nil {
				return errors.New("qstat: cannot query queue")
			}
		}
	*/
	ii := 0
	for index, _ := range retTable {
		if index == 0 {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", retTable[0][0], retTable[0][1], retTable[0][2], retTable[0][3], retTable[0][4], retTable[0][5], retTable[0][6])
			fmt.Printf("------------------------------------------------------------\n")
			continue
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\t%s\n", retTable[index][0], retTable[index][1], retTable[index][2], retTable[index][3], retTable[index][4], retTable[index][5], retTable[index][6])
		ii += 1
	}
	return nil

}

// TODO: fix
func QStat(args []string) error {
	statOptions, _, err := parseQStatArgs(args)
	if err != nil {
		log.Fatal(err)
		return err
	}
	if _, ok := statOptions["help"]; ok {
		if *statOptions["help"].(*bool) {
			qstatVersion()
			return nil
		}
	}
	if val, ok := statOptions["u"]; ok {
		return qstatUser(*val.(*string))
	}

	return nil
}

func qAcctPrintJob(number int, job core.JarviceJob) {
	// Only print HPC jobs that have completed
	if job.EndTime == 0 || len(job.ApiSubmission.Queue) == 0 {
		//if job.EndTime == 0 {
		return
	}
	fmt.Println("==============================================================")
	fmt.Printf("qname\t%s\n", job.ApiSubmission.Queue)
	fmt.Printf("hostname\t%s\n", job.ApiSubmission.Machine.Type)
	fmt.Printf("group\t%s\n", job.User)
	fmt.Printf("owner\t%s\n", job.User)
	fmt.Printf("jobname\t%s\n", job.Label)
	fmt.Printf("jobnumber\t%d\n", number)
	fmt.Printf("account\t%s\n", job.User)
	// Times are best effort
	subTime := time.Unix(int64(job.SubmitTime), 0)
	fmt.Printf("qsub_time\t%s\n", subTime.Format(time.UnixDate))
	startTime := time.Unix(int64(job.StartTime), 0)
	fmt.Printf("start_time\t%s\n", startTime.Format(time.UnixDate))
	endTime := time.Unix(int64(job.EndTime), 0)
	fmt.Printf("end_time\t%s\n", endTime.Format(time.UnixDate))
	qAcctFailed := 0
	if job.ExitCode != 0 {
		qAcctFailed = 1
	}
	fmt.Printf("failed\t%d\n", qAcctFailed)
	fmt.Printf("exit_status\t%d\n", job.ExitCode)
}

func QAcct(args []string) error {
	/*	acctOptions, _, err := parseQAcctArgs(args)
		if err != nil {
			log.Fatal(err)
			return err
		}
	*/
	// Read JARVICE config for selected cluster
	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("qacct: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	clusterName := "default"
	var cluster core.JarviceCluster
	if val, ok := myConfig[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("qacct: cannot find credentials for " + clusterName)
	}
	userCreds := cluster.Creds
	urlString := cluster.Endpoint + "/jarvice/jobs?completed=true&username=" + userCreds.Username + "&apikey=" + userCreds.Apikey
	res, err := http.Get(urlString)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("qacct: cannot query queue")
	}
	var jarviceJobs core.JarviceJobs
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("qacct: HTTP read error")
	}
	if err := json.Unmarshal([]byte(body), &jarviceJobs); err != nil {
		return errors.New("qacct: cannot read response")
	}
	for index, val := range jarviceJobs {
		qAcctPrintJob(index, val)
	}
	return nil
}

func QConf(args []string) error {
	var myConfig core.JarviceConfig
	if config, err := core.ReadJarviceConfig(); err != nil {
		return errors.New("qconf: cannot read JARVICE config")
	} else {
		myConfig = config
	}
	clusterName := "default"
	var cluster core.JarviceCluster
	if val, ok := myConfig[clusterName]; ok {
		cluster = val
	} else {
		return errors.New("qconf: cannot find credentials for " + clusterName)
	}
	userCreds := cluster.Creds
	urlString := cluster.Endpoint + "/jarvice/queues?username=" + userCreds.Username + "&apikey=" + userCreds.Apikey
	res, err := http.Get(urlString)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("qconf: cannot query queue")
	}
	jarviceQueues := []string{}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("qconf: HTTP read error")
	}
	if err := json.Unmarshal([]byte(body), &jarviceQueues); err != nil {
		return errors.New("qconf: cannot read response")
	}
	if len(jarviceQueues) < 1 {
		fmt.Println("default")
	} else {
		fmt.Printf("%s\n", strings.Join(jarviceQueues, "\n"))
	}
	return nil
}

// XXX
