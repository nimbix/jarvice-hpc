package jarvice

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/jessevdk/go-flags"
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
	Insecure bool         `json:"jarvice_insecure"`
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
	JobProject  *string            `json:"job_project,omitempty"`
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
	Name           string `json:"name"`
	App            string `json:"app"`
	DefaultMachine string `json:"machine"`
	MachineScale   int    `json:"size"`
}

type JarviceQueues = map[string]JarviceQueue

func setSecurePolicy(insecure bool) {
	if insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
}

// Submit job request to JARVICE API
func JarviceSubmitJob(url string, insecure bool, jobReq JarviceJobRequest) (JarviceJobResponse, error) {

	submitErrMsg := "core: JARVICE submit failed: "
	jsonBytes, err := json.Marshal(jobReq)
	if err != nil {
		return JarviceJobResponse{}, errors.New(submitErrMsg + "marshal JSON")
	}
	setSecurePolicy(insecure)
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

func HpcLogin(endpoint string, insecure bool, cluster, username, apikey,
	vault string) (err error) {
	config, _ := ReadJarviceConfig()
	var jarviceAPI string
	if req, herr := http.NewRequest("GET", endpoint, nil); herr != nil {
		err = fmt.Errorf("jarvice: unable to parse %s: %w", endpoint, herr)
		return
	} else {
		jarviceAPI = req.URL.String()
	}
	config[cluster] = JarviceCluster{
		Endpoint: jarviceAPI,
		Insecure: insecure,
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
	// set config TARGET (best effort)
	WriteJarviceConfigTarget(cluster)
	return
}

func HpcLive(cluster string) (err error) {
	config, _ := ReadJarviceConfig()
	if _, ok := config[cluster]; !ok {
		return fmt.Errorf("%s cluster does not exists", cluster)
	}
	if !testJarviceEndpoint(cluster, config) {
		err = errors.New("jarvice: JARVICE endpoint not live")
		return
	}
	fmt.Println(cluster, "is live")
	if !testJarviceCreds(cluster, config) {
		err = errors.New("jarvice: unable to validate JARVICE credentials")
		return
	}
	fmt.Println(config[cluster].Creds.Username, "logged in")
	return
}

func ApiReq(endpoint, api string, insecure bool,
	args url.Values) (body []byte, err error) {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, errors.New("Invalid URL endpoint")
	}
	if u == nil {
		return nil, fmt.Errorf("Invalid syntax %s", endpoint)
	}
	u.Path = path.Clean(u.Path + "/jarvice/" + api)
	u.RawQuery = args.Encode()
	setSecurePolicy(insecure)
	if resp, err := http.Get(u.String()); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, errors.New("HTTP IO error")
		} else {
			if resp.StatusCode != http.StatusOK {
				respMap := map[string]string{}
				json.Unmarshal(body, &respMap)
				errMsg := ""
				if msg, ok := respMap["error"]; ok {
					errMsg = msg
				}
				return nil, errors.New("API req /jarvice/" + api + ": " + errMsg)
			} else {
				return body, nil
			}
		}
	}
}

func testJarviceCreds(cluster string, config JarviceConfig) bool {
	// Test credential using JARVICE API endpoint that requires authorization
	if myCluster, ok := config[cluster]; ok {
		if _, err := ApiReq(myCluster.Endpoint, "machines", myCluster.Insecure,
			myCluster.GetUrlCreds()); err == nil {
			return true
		}
	}
	return false
}

func testJarviceEndpoint(cluster string, config JarviceConfig) bool {
	if myCluster, ok := config[cluster]; ok {
		if _, err := ApiReq(myCluster.Endpoint, "live", myCluster.Insecure,
			url.Values{}); err == nil {
			return true
		}
	}
	return false
}

func HpcVault(vault string) (err error) {

	config, err := ReadJarviceConfig()
	if err != nil {
		return errors.New("vault: config not found. Try login first")
	}
	cluster := ReadJarviceConfigTarget()
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

func ReadJarviceConfigTarget() string {
	// Best effort (default: "default")
	defaultTarget := "default"
	configPath := path.Dir(getJarviceConfigPath())
	filename := configPath + "/TARGET"
	if !fileExist(filename) {
		return defaultTarget
	}
	jsonFile, err := os.Open(filename)
	defer jsonFile.Close()
	if err != nil {
		return defaultTarget
	}
	bytes, _ := ioutil.ReadAll(jsonFile)
	return string(bytes)
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
						// strip off comments
						flagLine := strings.TrimLeft(strings.Split(line[len(directive)+1:], "#")[0], " ")
						elements := strings.Split(strings.TrimRight(flagLine, " "), " ")
						// go through elements in line to build args
						appendArg := false
						for _, element := range elements {
							// is element a flag (- or -- prefix)
							if ok := len(element) > 0; ok && element[0] == '-' {
								appendArg = false
								dict := strings.Split(element, "=")
								if len(dict) == 2 {
									args = append(args, dict[0])
									args = append(args, dict[1])
									appendArg = true
								} else {
									args = append(args, dict[0])
								}
							} else {
								if appendArg {
									index := len(args) - 1
									args[index] = args[index] + " " + element
								} else {
									appendArg = true
									args = append(args, element)
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

func GetClusterConfig() (cluster JarviceCluster, err error) {
	config, err := ReadJarviceConfig()
	if err != nil {
		return JarviceCluster{}, errors.New("cannot read JARVICE config")
	}
	clusterName := ReadJarviceConfigTarget()
	if val, ok := config[clusterName]; ok {
		return val, nil
	}
	return JarviceCluster{}, errors.New("cannot find credentials for " + clusterName)
}

func CreateHelpErr() error {
	err := flags.Error{
		Type:    flags.ErrHelp,
		Message: "show help message",
	}
	return &err
}

func PreprocessArgs(args []string) ([]string, error) {
	pArgs := []string{}
	for _, arg := range args {
		pArgs = append(pArgs, arg)
	}
	// strip path for arg 0
	pArgs[0] = filepath.Base(args[0])
	for index, val := range pArgs {
		if strings.HasPrefix(val, "-") && len(val[1:]) > 1 && val[1] != '-' {
			pArgs[index] = "-" + val
		}
	}
	switch pArgs[0] {
	case "qsub", JobScriptArg:
		fail := false
		// preprocess -pe <pe-name> <pe-int>
		// remove <pe-name>
		for index, val := range pArgs {
			if val == "-pe" || val == "--pe" {
				if len(pArgs) > index+1 {
					if test := strings.Split(pArgs[index+1], " "); len(test) == 2 {
						if _, check := strconv.ParseInt(test[1], 10, 64); check != nil {
							if len(pArgs) > index+2 {
								// -pe <pe-name> <pe-int>
								pArgs = append(pArgs[:index+1], pArgs[index+2:]...)
							} else {
								// missing <pe-int>
								fail = true
							}
						} else {
							// index+1 contains a string with int (e.g. "hpc 2")
							pArgs = append(pArgs[:index+1],
								append([]string{test[1]},
									pArgs[index+2:]...)...)
						}
					} else if len(pArgs) > index+2 {
						// -pe <pe-name> <pe-int>
						pArgs = append(pArgs[:index+1], pArgs[index+2:]...)
					} else {
						fail = true
					}
				} else {
					fail = true
				}
				if fail {
					return nil, errors.New("unable to preprocess qsub parallel environment\n" +
						"-pe <pe-name> <int>")
				}
			}
		}
	default:
		// do nothing
	}
	return pArgs, nil
}

func IsYes(str string) bool {
	if str == "y" || str == "Y" || str == "yes" || str == "Yes" {
		return true
	}
	return false
}

const JobScriptArg = "PARSE_JOBSCRIPT"

func ParseJobFlags(data interface{}, parser *flags.Parser,
	jobScriptParser *flags.Parser, args []string, override bool) error {

	pArgs, err := PreprocessArgs(args)
	if err != nil {
		return err
	}
	if _, err := jobScriptParser.ParseArgs(pArgs); err != nil ||
		jobScriptParser.Active == nil {
		return errors.New("unable to parse jobscript flags")
	}

	for _, option := range jobScriptParser.Active.Options() {
		if option.IsSet() && !option.IsSetDefault() {
			optionName := option.Field().Name
			optionValue := option.Value()
			// set flag in flags if not already set or override requested
			var activeOption *flags.Option
			if val := option.ShortName; val != 0 {
				activeOption = parser.Active.FindOptionByShortName(val)
			} else if val := option.LongName; len(val) > 0 {
				activeOption = parser.Active.FindOptionByLongName(val)
			}
			isSet := false
			if activeOption != nil {
				isSet = activeOption.IsSet() && !activeOption.IsSetDefault()
			}
			if !isSet || override {
				ps := reflect.ValueOf(data)
				s := ps.Elem()
				f := s.FieldByName(optionName)

				if f.IsValid() {
					if f.CanSet() {
						switch f.Kind() {
						case reflect.Int:
							x := int64(optionValue.(int))
							if !f.OverflowInt(x) {
								f.SetInt(x)
							}
						case reflect.String:
							f.SetString(optionValue.(string))
						case reflect.Bool:
							f.SetBool(optionValue.(bool))
						case reflect.Slice:
							switch o := optionValue.(type) {
							case []string:
								reflect.Copy(f, reflect.ValueOf(o))
							}
						default:
						}
					}
				}
			}
		}
	}

	return nil
}

func PrintTable(table [][]string, line bool) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()
	for index, record := range table {
		if index == 1 && line {
			for i := 0; i < len(record); i++ {
				for j := 0; j < int(math.Ceil(float64(len(record[i]))/8.0))*8; j++ {
					fmt.Fprintf(w, "%s", "-")
				}
				fmt.Fprintf(w, "\t")
			}
			fmt.Fprintf(w, "\n")
		}
		for _, value := range record {
			fmt.Fprintf(w, "%s\t", value)
		}
		fmt.Fprintf(w, "\n")
	}
}
