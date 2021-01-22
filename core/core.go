package jarvice

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
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
	JarviceHpcGeometry    = "1280x720"
	JarviceHpcStaging     = false
	JarviceHpcCheckedout  = false
	JarviceHpcCommandName = "HpcJob"
)

const JarviceHpcConfigEnv = "JARVICE_HPC_CONFIG"

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

type JarviceCluster struct {
	Endpoint string       `json:"jarvice_endpoint"`
	Vault    string       `json:"jarvice_vault"`
	Creds    JarviceCreds `json:"jarvice_user"`
}

func (c JarviceCluster) GetUrlCreds() url.Values {
	values := url.Values{}
	values.Add("username", c.Creds.Username)
	values.Add("apikey", c.Creds.Apikey)
	return values
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

type HpcReq struct {
	JobEnvConfig string            `json:"hpc_job_env_config"`
	JobScript    string            `json:"hpc_job_script"`
	JobShell     string            `json:"hpc_job_shell"`
	Queue        string            `json:"hpc_queue"`
	Umask        int               `json:"hpc_umask"`
	Envs         map[string]string `json:"hpc_envs"`
	Resources    map[string]string `json:"hpc_resources"`
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
	Hpc         HpcReq             `json:"hpc"`
	Licenses    *string            `json:"licenses,omitempty"`
}

// Return from API (jarvice/submit)
type JarviceJobResponse struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type JarviceApiSubmission struct {
	Machine JarviceMachine `json:"machine"`
	Queue   string         `json:"queue"`
}

type JarviceJob struct {
	Label         string               `json:"job_label"`
	User          string               `json:"job_owner_username"`
	Status        string               `json:"job_status"`
	SubmitTime    int                  `json:"job_submit_time"`
	StartTime     int                  `json:"job_start_time"`
	EndTime       int                  `json:"job_end_time"`
	ExitCode      int                  `json:"job_exitcode"`
	App           string               `json:"job_application"`
	ApiSubmission JarviceApiSubmission `json:"job_api_submission"`
}
type JarviceJobs = map[int]JarviceJob

type JarviceQueue struct {
	Name                string `json:"name"`
	App                 string `json:"app"`
	DefaultMachine      string `json:"machine"`
	DefaultMachineScale int    `json:"size"`
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
		body, _ := ioutil.ReadAll(resp.Body)
		errResp := map[string]string{}
		var errMsg string
		if err := json.Unmarshal([]byte(body), &errResp); err == nil {
			if msg, ok := errResp["error"]; ok {
				errMsg = ": " + msg
			}
		}
		return JarviceJobResponse{}, errors.New(submitErrMsg + http.StatusText(resp.StatusCode) + errMsg)
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

func HpcLogin(endpoint, username, apikey, cluster, vault string) (err error) {

	config, _ := ReadJarviceConfig()
	config[cluster] = JarviceCluster{
		Endpoint: endpoint,
		Vault:    vault,
		Creds: JarviceCreds{
			Username: username,
			Apikey:   apikey,
		},
	}
	if !testJarviceEndpoint(cluster, config) {
		err = errors.New("jarvice: JARVICE endpoint not live")
		return
	}
	if !testJarviceCreds(cluster, config) {
		err = errors.New("jarvice: unable to validate JARVICE credentials")
		return
	}
	err = WriteJarviceConfig(config)
	return
}

func ApiReq(endpoint, api string, args url.Values) (body []byte, err error) {
	u, _ := url.ParseRequestURI(endpoint)
	u.Path = path.Clean(u.Path + "/jarvice/" + api)
	u.RawQuery = args.Encode()
	if resp, err := http.Get(u.String()); err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	} else {
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		} else {
			return body, nil
		}
	}
}

func testJarviceCreds(cluster string, config JarviceConfig) bool {
	// Test credential using JARVICE API endpoint that requires authorization
	if myCluster, ok := config[cluster]; ok {
		if _, err := ApiReq(myCluster.Endpoint, "machines", myCluster.GetUrlCreds()); err == nil {
			return true
		}
	}
	return false
}

func testJarviceEndpoint(cluster string, config JarviceConfig) bool {
	if myCluster, ok := config[cluster]; ok {
		if _, err := ApiReq(myCluster.Endpoint, "live", url.Values{}); err == nil {
			return true
		}
	}
	return false
}

func HpcVault(cluster, vault string) (err error) {

	config, err := ReadJarviceConfig()
	if err != nil {
		return errors.New("vault: config not found. Try login first")
	}
	if myCluster, ok := config[cluster]; !ok {
		return errors.New("vault: config not found")
	} else {
		myCluster.Vault = vault
		config[cluster] = myCluster

		if err := WriteJarviceConfig(config); err != nil {
			return errors.New("vault: unable to write config file")
		}
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
			return JarviceHpcConfigFilename
		}
	}
	if _, err := os.Create(backupPath + JarviceHpcConfigFilename); err != nil {
		return JarviceHpcConfigFilename
	}
	return backupPath + JarviceHpcConfigFilename
}

func WriteJarviceConfigTarget(target string) error {
	configPath := path.Dir(getJarviceConfigPath())
	configFile := configPath + "/TARGET"
	// Ensure config file uses proper permissions
	// TODO: replace with perms check/error?
	os.Chmod(configFile, JarviceHpcConfigFilePerms)
	// XXX
	err := ioutil.WriteFile(configFile, []byte(target), JarviceHpcConfigFilePerms)
	return err
}

func ReadJarviceConfigTarget() (string, error) {
	configPath := path.Dir(getJarviceConfigPath())
	filename := configPath + "/TARGET"
	if !fileExist(filename) {
		return "", errors.New("cannot read JARVICE config TARGET")
	}
	jsonFile, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()
	bytes, _ := ioutil.ReadAll(jsonFile)

	return string(bytes), nil
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

	var scanner *bufio.Scanner

	if filename == "STDIN" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
			return JobScript{}, err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	shell := "/bin/sh"
	var args []string
	script := []byte{}

	shelled := false
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if len(line) > 1 {
			if line[0] == '#' {
				if line[1] == '!' && !shelled {
					shell = line[2:]
					shelled = true
					continue
				} else if line[1] != directive[0] {
					// noop for comment
					continue
				} else if len(line) > (len(directive) + 1) {
					if line[:len(directive)+1] == "#"+directive {
						parts := strings.Split(line[len(directive)+1:], "-")
						if len(parts) > 0 {
							for _, val := range parts[1:] {
								// strip off comments
								val = strings.Split(val, "#")[0]
								tempArgs := strings.Split(strings.TrimRight("-"+val, " "), " ")
								args = append(args, tempArgs[0])
								if len(tempArgs) > 1 {
									args = append(args, strings.Join(tempArgs[1:], " "))
								}
							}
						}
						continue
					}
				}
			}
		}
		script = append(script, scanner.Bytes()...)
		script = append(script, '\n')
	}
	return JobScript{
		Shell:  shell,
		Args:   args,
		Script: script,
	}, nil
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	// best effort
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	parts := strings.Split(localAddr, ":")
	return parts[0]
}
