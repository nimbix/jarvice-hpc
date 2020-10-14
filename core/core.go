package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	JarviceHpcConfigPath      = "/.config/jarvice-hpc/"
	JarviceHpcConfigFilename  = "config.json"
	JarviceHpcConfigFilePerms = 0600
)

// Default constants
const (
	// TODO: use application command specified by queue
	JarviceHpcCommand = "Batch"
	// XXX
	JarviceHpcGeometry   = "1280x720"
	JarviceHpcStaging    = false
	JarviceHpcCheckedout = false
)

const JarviceHpcConfigEnv = "JARVICE_HPC_CONFIG"

// TODO: get default queue from JARVICE API
func MyTestQueue() JarviceQueue {
	return JarviceQueue{
		Name:                "default",
		App:                 "khill-hpc_test",
		AppCommand:          "Batch",
		DefaultMachine:      "n3",
		DefaultMachineScale: 1,
	}
}

// XXX
// Data for HPC job script
/*
#!/bin/bash
#SBATCH --job-name=job_test    # Job name
#SBATCH --time=00:05:00
pwd; hostname; date
*/
type JobScript struct {
	Shell string `json:"hpc_shell"`
	// Args parsed from SBATCH directive
	Args   []string `json:"hpc_args"`
	Script []byte   `json:"hpc_script"`
}

type JobSpec struct {
	// Job environment variables
	SubmitDirectory string            `json:"hpc_submit_directory"`
	SubmitHost      string            `json:"hpc_submit_host"`
	QueueName       string            `json:"hpc_queue_name"`
	JobName         string            `json:"hpc_job_name"`
	UserEnv         map[string]string `json:"hpc_user_env"`

	Queue             string `json:"hpc_queue"`
	NodeCount         int    `json:"hpc_node_count"`
	CpuCount          int    `json:"hpc_cpu_count"`
	WallClockLimit    string `json:"hpc_wall_clock_limit"`
	OutputFile        string `json:"hpc_output_file"`
	ErrorFile         string `json:"hpc_error_file"`
	CopyEnvironment   string `json:"hpc_opy_environment"`
	EventNotification string `json:"hpc_event_notification"`
	EmailAddress      string `json:"hpc_email_address"`
	JobRestart        bool   `json:"hpc_job_restart"`
	WorkingDirectory  string `json:"hpc_working_directory"`
	Exclusive         bool   `json:"hpc_exclusive"`
	Memory            string `json:"hpc_memory"`
	ChargeAccount     string `json:"hpc_charge_account"`
	TasksPerNode      int    `json:"hpc_tasks_per_node"`
	CpusPerTask       int    `json:"hpc_cpus_per_task"`
	JobDependency     string `json:"hpc_job_dependency"`
	JobProject        string `json:"hpc_job_project"`
	GenericResources  string `json:"hpc_generic_resources"`
	Licenses          string `json:"hpc_licenses"`
	BeginTime         string `json:"hpc_begin_time"`
}

// Layout for JARVICE config file
/*
{
	"default": {
		"jarvice_endpoint": "<jarvice-api-url>",
		"jarvice_vault": "ephemeral",
		"jarvice_user": {
			"username": "<username>",
			"apikey": "<apikey>"
		}
	}
}
*/
type JarviceCluster struct {
	Endpoint string       `json:"jarvice_endpoint"`
	Vault    string       `json:"jarvice_vault"`
	Creds    JarviceCreds `json:"jarvice_user"`
}

type JarviceCreds struct {
	Username string `json:"username"`
	Apikey   string `json:"apikey"`
}

type JarviceConfig map[string]JarviceCluster

// JARVICE submission format
type JarviceApplication struct {
	Command    string                 `json:"command"`
	Walltime   string                 `json:"walltime,omitempty"`
	Geometry   string                 `json:"geometry"`
	Parameters map[string]interface{} `json:"parameters"`
}

type JarviceMachine struct {
	Type  string `json:"type"`
	Nodes int    `json:"nodes"`
}

type JarviceVault struct {
	Name     string `json:"name"`
	ReadOnly bool   `json:"readonly"`
	Force    bool   `json:"force"`
}

type JarviceJobRequest struct {
	App         string             `json:"app"`
	Staging     bool               `json:"staging"`
	Checkedout  bool               `json:"checkedout"`
	Application JarviceApplication `json:"application"`
	Machine     JarviceMachine     `json:"machine"`
	Vault       JarviceVault       `json:"vault"`
	JobLabel    string             `json:"job_label,omitempty"`
	User        JarviceCreds       `json:"user"`
}

// Return from API (jarvice/submit)
type JarviceJobResponse struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type JarviceQueue struct {
	Name                string `json:"hpc_queue_name"`
	App                 string `json:"hpc_queue_app"`
	AppCommand          string `json:"hpc_queue_app_command"`
	DefaultMachine      string `json:"hpc_queue_default_machine"`
	DefaultMachineScale int    `json:"hpc_queue_default_machine_scale"`
}

// Subcommands for 'jarvice' CLI command
func Jarvice(args []string) error {
	switch subCommand := args[0]; subCommand {
	case "login":
		return jarviceHpcLogin(args[1:])
	case "vault":
		return jarviceHpcVault(args[1:])
	default:
		fmt.Println("jarvice: unknown command")
	}
	return nil
}

// Submit job request to JARVICE API
func JarviceSubmitJob(url string, jobReq JarviceJobRequest) (JarviceJobResponse, error) {

	submitErrMsg := "core: JARVICE submit failed: "
	jsonBytes, err := json.Marshal(jobReq)
	if err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "marshal JSON")
	}

	req, err := http.NewRequest("POST", url+"/jarvice/submit",
		bytes.NewBuffer(jsonBytes))
	if err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "http request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "http client")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return JarviceJobResponse{}, errors.New(submitErrMsg + http.StatusText(resp.StatusCode))
	}

	var jarviceResponse JarviceJobResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "read response")
	}
	if err := json.Unmarshal([]byte(body), &jarviceResponse); err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "decode response")
	}

	return jarviceResponse, nil
}

func jarviceHpcLogin(args []string) (err error) {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)

	endpoint := flags.String("endpoint", "", "JARVICE API endpoint")
	username := flags.String("username", "", "JARVICE username")
	apikey := flags.String("apikey", "", "JARVICE apikey")
	cluster := flags.String("cluster", "default", "JARVICE cluster label")
	vault := flags.String("vault", "ephemeral", "default JARVICE vault")

	if flags.Parse(args) != nil {
		err = errors.New("jarvice: cannot process arguments")
		return
	}

	config := make(JarviceConfig)
	config, _ = ReadJarviceConfig()
	config[*cluster] = JarviceCluster{
		Endpoint: *endpoint,
		Vault:    *vault,
		Creds: JarviceCreds{
			Username: *username,
			Apikey:   *apikey,
		},
	}

	if !testJarviceEndpoint(*cluster, config) {
		err = errors.New("jarvice: JARVICE endpoint not live")
		return
	}
	if !testJarviceCreds(*cluster, config) {
		err = errors.New("jarvice: unable to validate JARVICE credentials")
		return
	}
	err = WriteJarviceConfig(config)
	return
}

func testJarviceCreds(cluster string, config JarviceConfig) bool {
	// Test credential using JARVICE API endpoint that requires authorization
	resp, err := http.Get(config[cluster].Endpoint + "/jarvice/machines" +
		"?username=" + config[cluster].Creds.Username +
		"&apikey=" + config[cluster].Creds.Apikey)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func testJarviceEndpoint(cluster string, config JarviceConfig) bool {
	resp, err := http.Get(config[cluster].Endpoint + "/jarvice/live")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func jarviceHpcVault(args []string) error {
	flags := flag.NewFlagSet("vault", flag.ContinueOnError)

	cluster := flags.String("cluster", "default", "JARVICE cluster label")
	vault := flags.String("vault", "ephemeral", "JARVICE vault")

	if flags.Parse(args) != nil {
		return errors.New("vault: cannot process arguments")
	}

	config, err := ReadJarviceConfig()
	if err != nil {
		return errors.New("vault: config not found. Try login first")
	}

	myCluster := config[*cluster]
	myCluster.Vault = *vault
	config[*cluster] = myCluster

	if err := WriteJarviceConfig(config); err != nil {
		return errors.New("vault: unable to write config file")
	}

	return nil
}

func fileExist(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Build path for config file
// Set from environment or use backup
// Use current directory as last resort
func getJarviceConfigPath() string {
	configPath := os.Getenv(JarviceHpcConfigEnv)
	if fileExist(configPath) {
		return configPath
	}
	backupPath := (os.Getenv("HOME") + JarviceHpcConfigPath)
	if fileExist(backupPath + JarviceHpcConfigFilename) {
		return backupPath + JarviceHpcConfigFilename
	} else {
		if err := os.MkdirAll(backupPath, 0744); err != nil {
			fmt.Println("test1")
			return JarviceHpcConfigFilename
		}
	}
	if _, err := os.Create(backupPath + JarviceHpcConfigFilename); err != nil {
		fmt.Println("test2")
		return JarviceHpcConfigFilename
	}
	return backupPath + JarviceHpcConfigFilename
}

func WriteJarviceConfig(config JarviceConfig) error {
	configFile := getJarviceConfigPath()
	file, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	// Ensure config file uses proper permissions
	// TODO: replace with perms check/error?
	os.Chmod(configFile, JarviceHpcConfigFilePerms)
	// XXX
	err = ioutil.WriteFile(configFile, file, JarviceHpcConfigFilePerms)
	return err
}

func ReadJarviceConfig() (JarviceConfig, error) {
	filename := getJarviceConfigPath()
	if !fileExist(filename) {
		return JarviceConfig{}, errors.New("cannot read JARVICE config")
	}
	jsonFile, err := os.Open(filename)
	if err != nil {
		return JarviceConfig{}, err
	}
	defer jsonFile.Close()
	bytes, _ := ioutil.ReadAll(jsonFile)

	var config JarviceConfig
	json.Unmarshal([]byte(bytes), &config)
	// Check if any cluster were found in config file
	if len(config) == 0 {
		return JarviceConfig{}, errors.New("invalid JARVICE config")
	}
	return config, nil
}

func ParseJobScript(directive, filename string) (JobScript, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return JobScript{}, err
	}
	defer file.Close()

	var shell string
	var args []string
	var script []byte

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	fmt.Println(scanner.Text())
	fmt.Println(scanner.Text()[:2])
	if line := scanner.Text(); line[:2] == "#!" {
		shell = line[2:]
		fmt.Println(shell)
	} else {
		shell = "/bin/sh"
	}
	fmt.Printf("#" + directive + "\n")
	parsed := false
	for scanner.Scan() {
		line := scanner.Text()
		if !parsed && line[:len(directive)+1] == "#"+directive {
			ss := strings.Fields(line[len(directive)+1:])[0]
			fmt.Println(ss)
			args = append(args, ss)
			continue
		} else {
			parsed = true
		}
		script = append(script, scanner.Bytes()...)
		script = append(script, '\n')
	}
	// TODO: remove
	fmt.Println("###########")
	fmt.Println(string(script))
	fmt.Println("###########")
	// XXX
	return JobScript{
		Shell:  shell,
		Args:   args,
		Script: script,
	}, nil
}
