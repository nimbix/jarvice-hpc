package slurm

import (
	"errors"

	flag "github.com/juju/gnuflag"
)

// Option descriptions
const (
	sBatchArrayDesc     = `Submit a job array, multiple jobs to be executed with identical parameters. The indexes specification identifies what array index values should be used. Multiple values may be specified using a comma separated list and/or a range of values with a "-" separator. For example, "--array=0-15" or "--array=0,6,16-32". A step function can also be specified with a suffix containing a colon and number. For example, "--array=0-15:4" is equivalent to "--array=0,4,8,12". A maximum number of simultaneously running tasks from the job array may be specified using a "%" separator. For example "--array=0-15%4" will limit the number of simultaneously running tasks from this job array to 4. The minimum index value is 0. the maximum value is one less than the configuration parameter MaxArraySize. NOTE: currently, federated job arrays only run on the local cluster. `
	sBatchAccountDesc   = `Charge resources used by this job to specified account. The account is an arbitrary string. The account name may be changed after job submission using the scontrol command.`
	sBatchAcctgFreqDesc = `Define the job accounting and profiling sampling intervals. This can be used to override the JobAcctGatherFrequency parameter in Slurm's configuration file, slurm.conf. The supported format is as follows:

        --acctg-freq=<datatype>=<interval>
            where <datatype>=<interval> specifies the task sampling interval for the jobacct_gather plugin or a sampling interval for a profiling type by the acct_gather_profile plugin. Multiple, comma-separated <datatype>=<interval> intervals may be specified. Supported datatypes are as follows:

                task=<interval>
                    where <interval> is the task sampling interval in seconds for the jobacct_gather plugins and for task profiling by the acct_gather_profile plugin. NOTE: This frequency is used to monitor memory usage. If memory limits are enforced the highest frequency a user can request is what is configured in the slurm.conf file. They can not turn it off (=0) either. 
                energy=<interval>
                    where <interval> is the sampling interval in seconds for energy profiling using the acct_gather_energy plugin 
                network=<interval>
                    where <interval> is the sampling interval in seconds for infiniband profiling using the acct_gather_interconnect plugin. 
                filesystem=<interval>
                    where <interval> is the sampling interval in seconds for filesystem profiling using the acct_gather_filesystem plugin. 


    The default value for the task sampling interval is 30 seconds. The default value for all other intervals is 0. An interval of 0 disables sampling of the specified type. If the task sampling interval is 0, accounting information is collected only at job termination (reducing Slurm interference with the job).

    Smaller (non-zero) values have a greater impact upon job performance, but a value of 30 seconds is not likely to be noticeable for applications having less than 10,000 tasks. `
	sBatchExtraNodeInfoDesc = `Restrict node selection to nodes with at least the specified number of sockets, cores per socket and/or threads per core. NOTE: These options do not specify the resource allocation size. Each value specified is considered a minimum. An asterisk (*) can be used as a placeholder indicating that all available resources of that type are to be utilized. Values can also be specified as min-max. The individual levels can also be specified in separate options if desired:

        --sockets-per-node=<sockets>
        --cores-per-socket=<cores>
        --threads-per-core=<threads>

    If task/affinity plugin is enabled, then specifying an allocation in this manner also results in subsequently launched tasks being bound to threads if the -B option specifies a thread count, otherwise an option of cores if a core count is specified, otherwise an option of sockets. If SelectType is configured to select/cons_res, it must have a parameter of CR_Core, CR_Core_Memory, CR_Socket, or CR_Socket_Memory for this option to be honored. If not specified, the scontrol show job will display 'ReqS:C:T=*:*:*'. This option applies to job allocations. `
	sBatchBatchDesc           = `Nodes can have features assigned to them by the Slurm administrator. Users can specify which of these features are required by their batch script using this options. For example a job's allocation may include both Intel Haswell and KNL nodes with features "haswell" and "knl" respectively. On such a configuration the batch script would normally benefit by executing on a faster Haswell node. This would be specified using the option "--batch=haswell". The specification can include AND and OR operators using the ampersand and vertical bar separators. For example: "--batch=haswell|broadwell" or "--batch=haswell|big_memory". The --batch argument must be a subset of the job's --constraint=<list> argument (i.e. the job can not request only KNL nodes, but require the script to execute on a Haswell node). If the request can not be satisfied from the resources allocated to the job, the batch script will execute on the first node of the job allocation. `
	sBatchBurstBufferDesc     = `Burst buffer specification. The form of the specification is system dependent. Note the burst buffer may not be accessible from a login node, but require that salloc spawn a shell on one of its allocated compute nodes. See the description of SallocDefaultCommand in the slurm.conf man page for more information about how to spawn a remote shell. `
	sBatchBurstBufferFileDesc = `Path of file containing burst buffer specification. The form of the specification is system dependent. These burst buffer directives will be inserted into the submitted batch script. `
	sBatchBeginDesc           = `Submit the batch script to the Slurm controller immediately, like normal, but tell the controller to defer the allocation of the job until the specified time.

    Time may be of the form HH:MM:SS to run a job at a specific time of day (seconds are optional). (If that time is already past, the next day is assumed.) You may also specify midnight, noon, fika (3 PM) or teatime (4 PM) and you can have a time-of-day suffixed with AM or PM for running in the morning or the evening. You can also say what day the job will be run, by specifying a date of the form MMDDYY or MM/DD/YY YYYY-MM-DD. Combine date and time using the following format YYYY-MM-DD[THH:MM[:SS]]. You can also give times like now + count time-units, where the time-units can be seconds (default), minutes, hours, days, or weeks and you can tell Slurm to run the job today with the keyword today and to run the job tomorrow with the keyword tomorrow. The value may be changed after job submission using the scontrol command. For example:

       --begin=16:00
       --begin=now+1hour
       --begin=now+60           (seconds by default)
       --begin=2010-01-20T12:34:00

        Notes on date/time specifications:
         - Although the 'seconds' field of the HH:MM:SS time specification is allowed by the code, note that the poll time of the Slurm scheduler is not precise enough to guarantee dispatch of the job on the exact second. The job will be eligible to start on the next poll following the specified time. The exact poll interval depends on the Slurm scheduler (e.g., 60 seconds with the default sched/builtin).
         - If no time (HH:MM:SS) is specified, the default is (00:00:00).
         - If a date is specified without a year (e.g., MM/DD) then the current year is assumed, unless the combination of MM/DD and HH:MM:SS has already passed for that year, in which case the next year is used. `
	sBatchClusterConstraintDesc = `Specifies features that a federated cluster must have to have a sibling job submitted to it. Slurm will attempt to submit a sibling job to a cluster if it has at least one of the specified features. If the "!" option is included, Slurm will attempt to submit a sibling job to a cluster that has none of the specified features. `
	sBatchCommentDesc           = `An arbitrary comment enclosed in double quotes if using spaces or some special characters. `
	sBatchConstraintDesc        = `Nodes can have features assigned to them by the Slurm administrator. Users can specify which of these features are required by their job using the constraint option. Only nodes having features matching the job constraints will be used to satisfy the request. Multiple constraints may be specified with AND, OR, matching OR, resource counts, etc. (some operators are not supported on all system types). Supported constraint options include:

        Single Name
            Only nodes which have the specified feature will be used. For example, --constraint="intel" 
        Node Count
            A request can specify the number of nodes needed with some feature by appending an asterisk and count after the feature name. For example, --nodes=16 --constraint="graphics*4 ..." indicates that the job requires 16 nodes and that at least four of those nodes must have the feature "graphics." 
        AND
            If only nodes with all of specified features will be used. The ampersand is used for an AND operator. For example, --constraint="intel&gpu" 
        OR
            If only nodes with at least one of specified features will be used. The vertical bar is used for an OR operator. For example, --constraint="intel|amd" 
        Matching OR
            If only one of a set of possible options should be used for all allocated nodes, then use the OR operator and enclose the options within square brackets. For example, --constraint="[rack1|rack2|rack3|rack4]" might be used to specify that all nodes must be allocated on a single rack of the cluster, but any of those four racks can be used. 
        Multiple Counts
            Specific counts of multiple resources may be specified by using the AND operator and enclosing the options within square brackets. For example, --constraint="[rack1*2&rack2*4]" might be used to specify that two nodes must be allocated from nodes with the feature of "rack1" and four nodes must be allocated from nodes with the feature "rack2".

            NOTE: This construct does not support multiple Intel KNL NUMA or MCDRAM modes. For example, while --constraint="[(knl&quad)*2&(knl&hemi)*4]" is not supported, --constraint="[haswell*2&(knl&hemi)*4]" is supported. Specification of multiple KNL modes requires the use of a heterogeneous job. 
        Brackets
            Brackets can be used to indicate that you are looking for a set of nodes with the different requirements contained within the brackets. For example, --constraint="[(rack1|rack2)*1&(rack3)*2]" will get you one node with either the "rack1" or "rack2" features and two nodes with the "rack3" feature. The same request without the brackets will try to find a single node that meets those requirements. 
        Parenthesis
            Parenthesis can be used to group like node features together. For example, --constraint="[(knl&snc4&flat)*4&haswell*1]" might be used to specify that four nodes with the features "knl", "snc4" and "flat" plus one node with the feature "haswell" are required. All options within parenthesis should be grouped with AND (e.g. "&") operands. 

`
	sBatchContiguousDesc = `If set, then the allocated nodes must form a contiguous set.

NOTE: If SelectPlugin=cons_res this option won't be honored with the topology/tree or topology/3d_torus plugins, both of which can modify the node ordering. 
`
	sBatchCoresPerSocketDesc = `Restrict node selection to nodes with at least the specified number of cores per socket. See additional information under -B option above when task/affinity plugin is enabled.`
	sBatchCpuFreqDesc        = `

    Request that job steps initiated by srun commands inside this sbatch script be run at some requested frequency if possible, on the CPUs selected for the step on the compute node(s).

    p1 can be [#### | low | medium | high | highm1] which will set the frequency scaling_speed to the corresponding value, and set the frequency scaling_governor to UserSpace. See below for definition of the values.

    p1 can be [Conservative | OnDemand | Performance | PowerSave] which will set the scaling_governor to the corresponding value. The governor has to be in the list set by the slurm.conf option CpuFreqGovernors.

    When p2 is present, p1 will be the minimum scaling frequency and p2 will be the maximum scaling frequency.

    p2 can be [#### | medium | high | highm1] p2 must be greater than p1.

    p3 can be [Conservative | OnDemand | Performance | PowerSave | UserSpace] which will set the governor to the corresponding value.

    If p3 is UserSpace, the frequency scaling_speed will be set by a power or energy aware scheduling strategy to a value between p1 and p2 that lets the job run within the site's power goal. The job may be delayed if p1 is higher than a frequency that allows the job to run within the goal.

    If the current frequency is < min, it will be set to min. Likewise, if the current frequency is > max, it will be set to max.

    Acceptable values at present include:

        ####
            frequency in kilohertz 
        Low
            the lowest available frequency 
        High
            the highest available frequency 
        HighM1
            (high minus one) will select the next highest available frequency 
        Medium
            attempts to set a frequency in the middle of the available range 
        Conservative
            attempts to use the Conservative CPU governor 
        OnDemand
            attempts to use the OnDemand CPU governor (the default value) 
        Performance
            attempts to use the Performance CPU governor 
        PowerSave
            attempts to use the PowerSave CPU governor 
        UserSpace
            attempts to use the UserSpace CPU governor 

    The following informational environment variable is set in the job step when --cpu-freq option is requested.

            SLURM_CPU_FREQ_REQ

    This environment variable can also be used to supply the value for the CPU frequency request if it is set when the 'srun' command is issued. The --cpu-freq on the command line will override the environment variable value. The form on the environment variable is the same as the command line. See the ENVIRONMENT VARIABLES section for a description of the SLURM_CPU_FREQ_REQ variable.

    NOTE: This parameter is treated as a request, not a requirement. If the job step's node does not support setting the CPU frequency, or the requested value is outside the bounds of the legal frequencies, an error is logged, but the job step is allowed to continue.

    NOTE: Setting the frequency for just the CPUs of the job step implies that the tasks are confined to those CPUs. If task confinement (i.e., TaskPlugin=task/affinity or TaskPlugin=task/cgroup with the "ConstrainCores" option) is not configured, this parameter is ignored.

    NOTE: When the step completes, the frequency and governor of each selected CPU is reset to the previous values.

    NOTE: When submitting jobs with the --cpu-freq option with linuxproc as the ProctrackType can cause jobs to run too quickly before Accounting is able to poll for job information. As a result not all of accounting information will be present.`
	sBatchCpusPerGpuDesc  = `Advise Slurm that ensuing job steps will require ncpus processors per allocated GPU. Not compatible with the --cpus-per-task option. `
	sBatchCpusPerTaskDesc = `Advise the Slurm controller that ensuing job steps will require ncpus number of processors per task. Without this option, the controller will just try to allocate one processor per task.

For instance, consider an application that has 4 tasks, each requiring 3 processors. If our cluster is comprised of quad-processors nodes and we simply ask for 12 processors, the controller might give us only 3 nodes. However, by using the --cpus-per-task=3 options, the controller knows that each task requires 3 processors on the same node, and the controller will grant an allocation of 4 nodes, one for each of the 4 tasks. `
	sBatchDeadlineDesc = `remove the job if no ending is possible before this deadline (start > (deadline - time[-min])). Default is no deadline. Valid time formats are:
HH:MM[:SS] [AM|PM]
MMDD[YY] or MM/DD[/YY] or MM.DD[.YY]
MM/DD[/YY]-HH:MM[:SS]
YYYY-MM-DD[THH:MM[:SS]]] `
	sBatchDelayBootDesc  = `Do not reboot nodes in order to satisfied this job's feature specification if the job has been eligible to run for less than this time period. If the job has waited for less than the specified period, it will use only nodes which already have the specified features. The argument is in units of minutes. A default value may be set by a system administrator using the delay_boot option of the SchedulerParameters configuration parameter in the slurm.conf file, otherwise the default value is zero (no delay). `
	sBatchDependencyDesc = `Defer the start of this job until the specified dependencies have been satisfied completed. <dependency_list> is of the form <type:job_id[:job_id][,type:job_id[:job_id]]> or <type:job_id[:job_id][?type:job_id[:job_id]]>. All dependencies must be satisfied if the "," separator is used. Any dependency may be satisfied if the "?" separator is used. Only one separator may be used. Many jobs can share the same dependency and these jobs may even belong to different users. The value may be changed after job submission using the scontrol command. Dependencies on remote jobs are allowed in a federation. Once a job dependency fails due to the termination state of a preceding job, the dependent job will never be run, even if the preceding job is requeued and has a different termination state in a subsequent execution.

    after:job_id[[+time][:jobid[+time]...]]
        After the specified jobs start or are cancelled and 'time' in minutes from job start or cancellation happens, this job can begin execution. If no 'time' is given then then there is no delay after start or cancellation. 
    afterany:job_id[:jobid...]
        This job can begin execution after the specified jobs have terminated. 
    afterburstbuffer:job_id[:jobid...]
        This job can begin execution after the specified jobs have terminated and any associated burst buffer stage out operations have completed. 
    aftercorr:job_id[:jobid...]
        A task of this job array can begin execution after the corresponding task ID in the specified job has completed successfully (ran to completion with an exit code of zero). 
    afternotok:job_id[:jobid...]
        This job can begin execution after the specified jobs have terminated in some failed state (non-zero exit code, node failure, timed out, etc). 
    afterok:job_id[:jobid...]
        This job can begin execution after the specified jobs have successfully executed (ran to completion with an exit code of zero). 
    expand:job_id
        Resources allocated to this job should be used to expand the specified job. The job to expand must share the same QOS (Quality of Service) and partition. Gang scheduling of resources in the partition is also not supported. "expand" is not allowed for jobs that didn't originate on the same cluster as the submitted job. 
    singleton
        This job can begin execution after any previously launched jobs sharing the same job name and user have terminated. In other words, only one job by that name and owned by that user can be running or suspended at any point in time. In a federation, a singleton dependency must be fulfilled on all clusters unless DependencyParameters=disable_remote_singleton is used in slurm.conf. 

`
	sBatchChdirDesc     = `Set the working directory of the batch script to directory before it is executed. The path can be specified as full path or relative path to the directory where the command is executed. `
	sBatchErrorDesc     = `Instruct Slurm to connect the batch script's standard error directly to the file name specified in the "filename pattern". By default both standard output and standard error are directed to the same file. For job arrays, the default file name is "slurm-%A_%a.out", "%A" is replaced by the job ID and "%a" with the array index. For other jobs, the default file name is "slurm-%j.out", where the "%j" is replaced by the job ID. See the filename pattern section below for filename specification options. `
	sBatchExclusiveDesc = `The job allocation can not share nodes with other running jobs (or just other users with the "=user" option or with the "=mcs" option). The default shared/exclusive behavior depends on system configuration and the partition's OverSubscribe option takes precedence over the job's option. `
	sBatchExportDesc    = `Identify which environment variables from the submission environment are propagated to the launched application. Note that SLURM_* variables are always propagated.

    --export=ALL

        Default mode if --export is not specified. All of the users environment will be loaded (either from callers environment or clean environment if --get-user-env is specified). 
    --export=NONE

        Only SLURM_* variables from the user environment will be defined. User must use absolute path to the binary to be executed that will define the environment. User can not specify explicit environment variables with NONE. --get-user-env will be ignored.
        This option is particularly important for jobs that are submitted on one cluster and execute on a different cluster (e.g. with different paths). To avoid steps inheriting environment export settings (e.g. NONE) from sbatch command, the environment variable SLURM_EXPORT_ENV should be set to ALL in the job script. 
    --export=<[ALL,]environment variables>

        Exports all SLURM_* environment variables along with explicitly defined variables. Multiple environment variable names should be comma separated. Environment variable names may be specified to propagate the current value (e.g. "--export=EDITOR") or specific values may be exported (e.g. "--export=EDITOR=/bin/emacs"). If ALL is specified, then all user environment variables will be loaded and will take precedence over any explicitly given environment variables.

            Example: --export=EDITOR,ARG1=test

                In this example, the propagated environment will only contain the variable EDITOR from the user's environment, SLURM_* environment variables, and ARG1=test. 
            Example: --export=ALL,EDITOR=/bin/emacs

                There are two possible outcomes for this example. If the caller has the EDITOR environment variable defined, then the job's environment will inherit the variable from the caller's environment. If the caller doesn't have an environment variable defined for EDITOR, then the job's environment will use the value given by --export. 

`
	sBatchExportFileDesc = `If a number between 3 and OPEN_MAX is specified as the argument to this option, a readable file descriptor will be assumed (STDIN and STDOUT are not supported as valid arguments). Otherwise a filename is assumed. Export environment variables defined in <filename> or read from <fd> to the job's execution environment. The content is one or more environment variable definitions of the form NAME=value, each separated by a null character. This allows the use of special characters in environment definitions. `
	sBatchNodefileDesc   = `Much like --nodelist, but the list is contained in a file of name node file. The node names of the list may also span multiple lines in the file. Duplicate node names in the file will be ignored. The order of the node names in the list is not important; the node names will be sorted by Slurm. `
	sBatchGetUserEnvDesc = `This option will tell sbatch to retrieve the login environment variables for the user specified in the --uid option. The environment variables are retrieved by running something of this sort "su - <username> -c /usr/bin/env" and parsing the output. Be aware that any environment variables already set in sbatch's environment will take precedence over any environment variables in the user's login environment. Clear any environment variables before calling sbatch that you do not want propagated to the spawned program. The optional timeout value is in seconds. Default value is 8 seconds. The optional mode value control the "su" options. With a mode value of "S", "su" is executed without the "-" option. With a mode value of "L", "su" is executed with the "-" option, replicating the login environment. If mode not specified, the mode established at Slurm build time is used. Example of use include "--get-user-env", "--get-user-env=10" "--get-user-env=10L", and "--get-user-env=S". `
	sBatchGidDesc        = `If sbatch is run as root, and the --gid option is used, submit the job with group's group access permissions. group may be the group name or the numerical group ID. `
	sBatchGpusDesc       = `Specify the total number of GPUs required for the job. An optional GPU type specification can be supplied. For example "--gpus=volta:3". Multiple options can be requested in a comma separated list, for example: "--gpus=volta:3,kepler:1". See also the --gpus-per-node, --gpus-per-socket and --gpus-per-task options. `
	sBatchGpuBindDesc    = `Bind tasks to specific GPUs. By default every spawned task can access every GPU allocated to the job.

Supported type options:

    closest
        Bind each task to the GPU(s) which are closest. In a NUMA environment, each task may be bound to more than one GPU (i.e. all GPUs in that NUMA environment). 
    map_gpu:<list>
        Bind by setting GPU masks on tasks (or ranks) as specified where <list> is <gpu_id_for_task_0>,<gpu_id_for_task_1>,... GPU IDs are interpreted as decimal values unless they are preceded with '0x' in which case they interpreted as hexadecimal values. If the number of tasks (or ranks) exceeds the number of elements in this list, elements in the list will be reused as needed starting from the beginning of the list. To simplify support for large task counts, the lists may follow a map with an asterisk and repetition count. For example "map_gpu:0*4,1*4". If the task/cgroup plugin is used and ConstrainDevices is set in cgroup.conf, then the GPU IDs are zero-based indexes relative to the GPUs allocated to the job (e.g. the first GPU is 0, even if the global ID is 3). Otherwise, the GPU IDs are global IDs, and all GPUs on each node in the job should be allocated for predictable binding results. 
    mask_gpu:<list>
        Bind by setting GPU masks on tasks (or ranks) as specified where <list> is <gpu_mask_for_task_0>,<gpu_mask_for_task_1>,... The mapping is specified for a node and identical mapping is applied to the tasks on every node (i.e. the lowest task ID on each node is mapped to the first mask specified in the list, etc.). GPU masks are always interpreted as hexadecimal values but can be preceded with an optional '0x'. To simplify support for large task counts, the lists may follow a map with an asterisk and repetition count. For example "mask_gpu:0x0f*4,0xf0*4". If the task/cgroup plugin is used and ConstrainDevices is set in cgroup.conf, then the GPU IDs are zero-based indexes relative to the GPUs allocated to the job (e.g. the first GPU is 0, even if the global ID is 3). Otherwise, the GPU IDs are global IDs, and all GPUs on each node in the job should be allocated for predictable binding results. 

`
	sBatchGpuFreqDesc = `Request that GPUs allocated to the job are configured with specific frequency values. This option can be used to independently configure the GPU and its memory frequencies. After the job is completed, the frequencies of all affected GPUs will be reset to the highest possible values. In some cases, system power caps may override the requested values. The field type can be "memory". If type is not specified, the GPU frequency is implied. The value field can either be "low", "medium", "high", "highm1" or a numeric value in megahertz (MHz). If the specified numeric value is not possible, a value as close as possible will be used. See below for definition of the values. The verbose option causes current GPU frequency information to be logged. Examples of use include "--gpu-freq=medium,memory=high" and "--gpu-freq=450".

Supported value definitions:

    low
        the lowest available frequency. 
    medium
        attempts to set a frequency in the middle of the available range. 
    high
        the highest available frequency. 
    highm1
        (high minus one) will select the next highest available frequency. 

`
	sBatchGpusPerNodeDesc   = `Specify the number of GPUs required for the job on each node included in the job's resource allocation. An optional GPU type specification can be supplied. For example "--gpus-per-node=volta:3". Multiple options can be requested in a comma separated list, for example: "--gpus-per-node=volta:3,kepler:1". See also the --gpus, --gpus-per-socket and --gpus-per-task options. `
	sBatchGpusPerSocketDesc = `Specify the number of GPUs required for the job on each socket included in the job's resource allocation. An optional GPU type specification can be supplied. For example "--gpus-per-socket=volta:3". Multiple options can be requested in a comma separated list, for example: "--gpus-per-socket=volta:3,kepler:1". Requires job to specify a sockets per node count ( --sockets-per-node). See also the --gpus, --gpus-per-node and --gpus-per-task options. `
	sBatchGpusPerTaskDesc   = `Specify the number of GPUs required for the job on each task to be spawned in the job's resource allocation. An optional GPU type specification can be supplied. For example "--gpus-per-task=volta:1". Multiple options can be requested in a comma separated list, for example: "--gpus-per-task=volta:3,kepler:1". See also the --gpus, --gpus-per-socket and --gpus-per-node options. This option requires an explicit task count, e.g. -n, --ntasks or "--gpus=X --gpus-per-task=Y" rather than an ambiguous range of nodes with -N, --nodes.
NOTE: This option will not have any impact on GPU binding, specifically it won't limit the number of devices set for CUDA_VISIBLE_DEVICES. `
	sBatchGresDesc      = `Specifies a comma delimited list of generic consumable resources. The format of each entry on the list is "name[[:type]:count]". The name is that of the consumable resource. The count is the number of those resources with a default value of 1. The count can have a suffix of "k" or "K" (multiple of 1024), "m" or "M" (multiple of 1024 x 1024), "g" or "G" (multiple of 1024 x 1024 x 1024), "t" or "T" (multiple of 1024 x 1024 x 1024 x 1024), "p" or "P" (multiple of 1024 x 1024 x 1024 x 1024 x 1024). The specified resources will be allocated to the job on each node. The available generic consumable resources is configurable by the system administrator. A list of available generic consumable resources will be printed and the command will exit if the option argument is "help". Examples of use include "--gres=gpu:2,mic:1", "--gres=gpu:kepler:2", and "--gres=help". `
	sBatchGresFlagsDesc = `Specify generic resource task binding options.

    disable-binding
        Disable filtering of CPUs with respect to generic resource locality. This option is currently required to use more CPUs than are bound to a GRES (i.e. if a GPU is bound to the CPUs on one socket, but resources on more than one socket are required to run the job). This option may permit a job to be allocated resources sooner than otherwise possible, but may result in lower job performance.
        NOTE: This option is specific to SelectType=cons_res. 
    enforce-binding
        The only CPUs available to the job will be those bound to the selected GRES (i.e. the CPUs identified in the gres.conf file will be strictly enforced). This option may result in delayed initiation of a job. For example a job requiring two GPUs and one CPU will be delayed until both GPUs on a single socket are available rather than using GPUs bound to separate sockets, however, the application performance may be improved due to improved communication speed. Requires the node to be configured with more than one socket and resource filtering will be performed on a per-socket basis.
        NOTE: This option is specific to SelectType=cons_tres. 

`
	sBatchHoldDesc = `Specify the job is to be submitted in a held state (priority of zero). A held job can now be released using scontrol to reset its priority (e.g. "scontrol release <job_id>"). `
	sBatchHintDesc = `Bind tasks according to application hints.

    compute_bound
        Select settings for compute bound applications: use all cores in each socket, one thread per core. 
    memory_bound
        Select settings for memory bound applications: use only one core in each socket, one thread per core. 
    [no]multithread
        [don't] use extra threads with in-core multi-threading which can benefit communication intensive applications. Only supported with the task/affinity plugin. 
    help
        show this help message 

`
	sBatchIgnorePbsDesc = `Ignore all "#PBS" and "#BSUB" options specified in the batch script. `
	sBatchInputDesc     = `Instruct Slurm to connect the batch script's standard input directly to the file name specified in the "filename pattern".

By default, "/dev/null" is open on the batch script's standard input and both standard output and standard error are directed to a file of the name "slurm-%j.out", where the "%j" is replaced with the job allocation number, as described below in the filename pattern section. `
	sBatchJobNameDesc = `Specify a name for the job allocation. The specified name will appear along with the job id number when querying running jobs on the system. The default is the name of the batch script, or just "sbatch" if the script is read on sbatch's standard input. `
	sBatchNoKillDesc  = `Do not automatically terminate a job if one of the nodes it has been allocated fails. The user will assume the responsibilities for fault-tolerance should a node fail. When there is a node failure, any active job steps (usually MPI jobs) on that node will almost certainly suffer a fatal error, but with --no-kill, the job allocation will not be revoked so the user may launch new job steps on the remaining nodes in their allocation.

Specify an optional argument of "off" disable the effect of the SBATCH_NO_KILL environment variable.

By default Slurm terminates the entire job allocation if any node fails in its range of allocated nodes. `
	sBatchKillOnInvalidDepDesc = `If a job has an invalid dependency and it can never run this parameter tells Slurm to terminate it or not. A terminated job state will be JOB_CANCELLED. If this option is not specified the system wide behavior applies. By default the job stays pending with reason DependencyNeverSatisfied or if the kill_invalid_depend is specified in slurm.conf the job is terminated. `
	sBatchLicensesDesc         = `Specification of licenses (or other resources available on all nodes of the cluster) which must be allocated to this job. License names can be followed by a colon and count (the default count is one). Multiple license names should be comma separated (e.g. "--licenses=foo:4,bar"). To submit jobs using remote licenses, those served by the slurmdbd, specify the name of the server providing the licenses. For example "--license=nastran@slurmdb:12". `
	sBatchClustersDesc         = `Clusters to issue commands to. Multiple cluster names may be comma separated. The job will be submitted to the one cluster providing the earliest expected job initiation time. The default value is the current cluster. A value of 'all' will query to run on all clusters. Note the --export option to control environment variables exported between clusters. Note that the SlurmDBD must be up for this option to work properly. `
	sBatchDistributionDesc     = ` Specify alternate distribution methods for remote processes. In sbatch, this only sets environment variables that will be used by subsequent srun requests. This option controls the assignment of tasks to the nodes on which resources have been allocated, and the distribution of those resources to tasks for binding (task affinity). The first distribution method (before the ":") controls the distribution of resources across nodes. The optional second distribution method (after the ":") controls the distribution of resources across sockets within a node. Note that with select/cons_res, the number of cpus allocated on each socket and node may be different. Refer to the mc_support document for more information on resource allocation, assignment of tasks to nodes, and binding of tasks to CPUs.

    First distribution method:

    block
        The block distribution method will distribute tasks to a node such that consecutive tasks share a node. For example, consider an allocation of three nodes each with two cpus. A four-task block distribution request will distribute those tasks to the nodes with tasks one and two on the first node, task three on the second node, and task four on the third node. Block distribution is the default behavior if the number of tasks exceeds the number of allocated nodes. 
    cyclic
        The cyclic distribution method will distribute tasks to a node such that consecutive tasks are distributed over consecutive nodes (in a round-robin fashion). For example, consider an allocation of three nodes each with two cpus. A four-task cyclic distribution request will distribute those tasks to the nodes with tasks one and four on the first node, task two on the second node, and task three on the third node. Note that when SelectType is select/cons_res, the same number of CPUs may not be allocated on each node. Task distribution will be round-robin among all the nodes with CPUs yet to be assigned to tasks. Cyclic distribution is the default behavior if the number of tasks is no larger than the number of allocated nodes. 
    plane
        The tasks are distributed in blocks of a specified size. The number of tasks distributed to each node is the same as for cyclic distribution, but the taskids assigned to each node depend on the plane size. Additional distribution specifications cannot be combined with this option. For more details (including examples and diagrams), please see
        the mc_support document
        and
        https://slurm.schedmd.com/dist_plane.html 
    arbitrary
        The arbitrary method of distribution will allocate processes in-order as listed in file designated by the environment variable SLURM_HOSTFILE. If this variable is listed it will override any other method specified. If not set the method will default to block. Inside the hostfile must contain at minimum the number of hosts requested and be one per line or comma separated. If specifying a task count (-n, --ntasks=<number>), your tasks will be laid out on the nodes in the order of the file.
        NOTE: The arbitrary distribution option on a job allocation only controls the nodes to be allocated to the job and not the allocation of CPUs on those nodes. This option is meant primarily to control a job step's task layout in an existing job allocation for the srun command.

    Second distribution method:
    block
        The block distribution method will distribute tasks to sockets such that consecutive tasks share a socket. 
    cyclic
        The cyclic distribution method will distribute tasks to sockets such that consecutive tasks are distributed over consecutive sockets (in a round-robin fashion). Tasks requiring more than one CPU will have all of those CPUs allocated on a single socket if possible. 
    fcyclic
        The fcyclic distribution method will distribute tasks to sockets such that consecutive tasks are distributed over consecutive sockets (in a round-robin fashion). Tasks requiring more than one CPU will have each CPUs allocated in a cyclic fashion across sockets. 

`
	sBatchMailTypeDesc = `Notify user by email when certain event types occur. Valid type values are NONE, BEGIN, END, FAIL, REQUEUE, ALL (equivalent to BEGIN, END, FAIL, REQUEUE, and STAGE_OUT), STAGE_OUT (burst buffer stage out and teardown completed), TIME_LIMIT, TIME_LIMIT_90 (reached 90 percent of time limit), TIME_LIMIT_80 (reached 80 percent of time limit), TIME_LIMIT_50 (reached 50 percent of time limit) and ARRAY_TASKS (send emails for each array task). Multiple type values may be specified in a comma separated list. The user to be notified is indicated with --mail-user. Unless the ARRAY_TASKS option is specified, mail notifications on job BEGIN, END and FAIL apply to a job array as a whole rather than generating individual email messages for each task in the job array. `
	sBatchMailUserDesc = `User to receive email notification of state changes as defined by --mail-type. The default value is the submitting user. `
	sBatchMcsLabelDesc = `Used only when the mcs/group plugin is enabled. This parameter is a group among the groups of the user. Default value is calculated by the Plugin mcs if it's enabled. `
	sBatchMemDesc      = `Specify the real memory required per node. Default units are megabytes unless the SchedulerParameters configuration parameter includes the "default_gbytes" option for gigabytes. Different units can be specified using the suffix [K|M|G|T]. Default value is DefMemPerNode and the maximum value is MaxMemPerNode. If configured, both parameters can be seen using the scontrol show config command. This parameter would generally be used if whole nodes are allocated to jobs (SelectType=select/linear). Also see --mem-per-cpu and --mem-per-gpu. The --mem, --mem-per-cpu and --mem-per-gpu options are mutually exclusive. If --mem, --mem-per-cpu or --mem-per-gpu are specified as command line arguments, then they will take precedence over the environment.

NOTE: A memory size specification of zero is treated as a special case and grants the job access to all of the memory on each node. If the job is allocated multiple nodes in a heterogeneous cluster, the memory limit on each node will be that of the node in the allocation with the smallest memory size (same limit will apply to every node in the job's allocation).

NOTE: Enforcement of memory limits currently relies upon the task/cgroup plugin or enabling of accounting, which samples memory use on a periodic basis (data need not be stored, just collected). In both cases memory use is based upon the job's Resident Set Size (RSS). A task may exceed the memory limit until the next periodic accounting sample. `
	sBatchMemPerCpuDesc = `Minimum memory required per allocated CPU. Default units are megabytes unless the SchedulerParameters configuration parameter includes the "default_gbytes" option for gigabytes. The default value is DefMemPerCPU and the maximum value is MaxMemPerCPU (see exception below). If configured, both parameters can be seen using the scontrol show config command. Note that if the job's --mem-per-cpu value exceeds the configured MaxMemPerCPU, then the user's limit will be treated as a memory limit per task; --mem-per-cpu will be reduced to a value no larger than MaxMemPerCPU; --cpus-per-task will be set and the value of --cpus-per-task multiplied by the new --mem-per-cpu value will equal the original --mem-per-cpu value specified by the user. This parameter would generally be used if individual processors are allocated to jobs (SelectType=select/cons_res). If resources are allocated by core, socket, or whole nodes, then the number of CPUs allocated to a job may be higher than the task count and the value of --mem-per-cpu should be adjusted accordingly. Also see --mem and --mem-per-gpu. The --mem, --mem-per-cpu and --mem-per-gpu options are mutually exclusive.

NOTE: If the final amount of memory requested by a job can't be satisfied by any of the nodes configured in the partition, the job will be rejected. This could happen if --mem-per-cpu is used with the --exclusive option for a job allocation and --mem-per-cpu times the number of CPUs on a node is greater than the total memory of that node. `
	sBatchMemPerGpuDesc = `Minimum memory required per allocated GPU. Default units are megabytes unless the SchedulerParameters configuration parameter includes the "default_gbytes" option for gigabytes. Different units can be specified using the suffix [K|M|G|T]. Default value is DefMemPerGPU and is available on both a global and per partition basis. If configured, the parameters can be seen using the scontrol show config and scontrol show partition commands. Also see --mem. The --mem, --mem-per-cpu and --mem-per-gpu options are mutually exclusive. `
	sBatchMemBindDesc   = `Bind tasks to memory. Used only when the task/affinity plugin is enabled and the NUMA memory functions are available. Note that the resolution of CPU and memory binding may differ on some architectures. For example, CPU binding may be performed at the level of the cores within a processor while memory binding will be performed at the level of nodes, where the definition of "nodes" may differ from system to system. By default no memory binding is performed; any task using any CPU can use any memory. This option is typically used to ensure that each task is bound to the memory closest to its assigned CPU. The use of any type other than "none" or "local" is not recommended.

NOTE: To have Slurm always report on the selected memory binding for all commands executed in a shell, you can enable verbose mode by setting the SLURM_MEM_BIND environment variable value to "verbose".

The following informational environment variables are set when --mem-bind is in use:

        SLURM_MEM_BIND_LIST
        SLURM_MEM_BIND_PREFER
        SLURM_MEM_BIND_SORT
        SLURM_MEM_BIND_TYPE
        SLURM_MEM_BIND_VERBOSE

See the ENVIRONMENT VARIABLES section for a more detailed description of the individual SLURM_MEM_BIND* variables.

Supported options include:

    help
        show this help message 
    local
        Use memory local to the processor in use 
    map_mem:<list>
        Bind by setting memory masks on tasks (or ranks) as specified where <list> is <numa_id_for_task_0>,<numa_id_for_task_1>,... The mapping is specified for a node and identical mapping is applied to the tasks on every node (i.e. the lowest task ID on each node is mapped to the first ID specified in the list, etc.). NUMA IDs are interpreted as decimal values unless they are preceded with '0x' in which case they interpreted as hexadecimal values. If the number of tasks (or ranks) exceeds the number of elements in this list, elements in the list will be reused as needed starting from the beginning of the list. To simplify support for large task counts, the lists may follow a map with an asterisk and repetition count. For example "map_mem:0x0f*4,0xf0*4". For predictable binding results, all CPUs for each node in the job should be allocated to the job. 
    mask_mem:<list>
        Bind by setting memory masks on tasks (or ranks) as specified where <list> is <numa_mask_for_task_0>,<numa_mask_for_task_1>,... The mapping is specified for a node and identical mapping is applied to the tasks on every node (i.e. the lowest task ID on each node is mapped to the first mask specified in the list, etc.). NUMA masks are always interpreted as hexadecimal values. Note that masks must be preceded with a '0x' if they don't begin with [0-9] so they are seen as numerical values. If the number of tasks (or ranks) exceeds the number of elements in this list, elements in the list will be reused as needed starting from the beginning of the list. To simplify support for large task counts, the lists may follow a mask with an asterisk and repetition count. For example "mask_mem:0*4,1*4". For predictable binding results, all CPUs for each node in the job should be allocated to the job. 
    no[ne]
        don't bind tasks to memory (default) 
    p[refer]
        Prefer use of first specified NUMA node, but permit
         use of other available NUMA nodes. 
    q[uiet]
        quietly bind before task runs (default) 
    rank
        bind by task rank (not recommended) 
    sort
        sort free cache pages (run zonesort on Intel KNL nodes) 
    v[erbose]
        verbosely report binding before task runs 

`
	sBatchMincpusDesc = `Specify a minimum number of logical cpus/processors per node. `
	sBatchNodesDesc   = `Request that a minimum of minnodes nodes be allocated to this job. A maximum node count may also be specified with maxnodes. If only one number is specified, this is used as both the minimum and maximum node count. The partition's node limits supersede those of the job. If a job's node limits are outside of the range permitted for its associated partition, the job will be left in a PENDING state. This permits possible execution at a later time, when the partition limit is changed. If a job node limit exceeds the number of nodes configured in the partition, the job will be rejected. Note that the environment variable SLURM_JOB_NODES will be set to the count of nodes actually allocated to the job. See the ENVIRONMENT VARIABLES section for more information. If -N is not specified, the default behavior is to allocate enough nodes to satisfy the requirements of the -n and -c options. The job will be allocated as many nodes as possible within the range specified and without delaying the initiation of the job. The node count specification may include a numeric value followed by a suffix of "k" (multiplies numeric value by 1,024) or "m" (multiplies numeric value by 1,048,576). `
	sBatchNtasksDesc  = `sbatch does not launch tasks, it requests an allocation of resources and submits a batch script. This option advises the Slurm controller that job steps run within the allocation will launch a maximum of number tasks and to provide for sufficient resources. The default is one task per node, but note that the --cpus-per-task option will change this default. `
	sBatchNetworkDesc = `    Specify information pertaining to the switch or network. The interpretation of type is system dependent. This option is supported when running Slurm on a Cray natively. It is used to request using Network Performance Counters. Only one value per request is valid. All options are case in-sensitive. In this configuration supported values include:

        system
            Use the system-wide network performance counters. Only nodes requested will be marked in use for the job allocation. If the job does not fill up the entire system the rest of the nodes are not able to be used by other jobs using NPC, if idle their state will appear as PerfCnts. These nodes are still available for other jobs not using NPC. 
        blade
            Use the blade network performance counters. Only nodes requested will be marked in use for the job allocation. If the job does not fill up the entire blade(s) allocated to the job those blade(s) are not able to be used by other jobs using NPC, if idle their state will appear as PerfCnts. These nodes are still available for other jobs not using NPC. 



    In all cases the job allocation request must specify the --exclusive option. Otherwise the request will be denied.



    Also with any of these options steps are not allowed to share blades, so resources would remain idle inside an allocation if the step running on a blade does not take up all the nodes on the blade.



    The network option is also supported on systems with IBM's Parallel Environment (PE). See IBM's LoadLeveler job command keyword documentation about the keyword "network" for more information. Multiple values may be specified in a comma separated list. All options are case in-sensitive. Supported values include:

        BULK_XFER[=<resources>]
            Enable bulk transfer of data using Remote Direct-Memory Access (RDMA). The optional resources specification is a numeric value which can have a suffix of "k", "K", "m", "M", "g" or "G" for kilobytes, megabytes or gigabytes. NOTE: The resources specification is not supported by the underlying IBM infrastructure as of Parallel Environment version 2.2 and no value should be specified at this time. 
        CAU=<count>
            Number of Collective Acceleration Units (CAU) required. Applies only to IBM Power7-IH processors. Default value is zero. Independent CAU will be allocated for each programming interface (MPI, LAPI, etc.) 
        DEVNAME=<name>
            Specify the device name to use for communications (e.g. "eth0" or "mlx4_0"). 
        DEVTYPE=<type>
            Specify the device type to use for communications. The supported values of type are: "IB" (InfiniBand), "HFI" (P7 Host Fabric Interface), "IPONLY" (IP-Only interfaces), "HPCE" (HPC Ethernet), and "KMUX" (Kernel Emulation of HPCE). The devices allocated to a job must all be of the same type. The default value depends upon depends upon what hardware is available and in order of preferences is IPONLY (which is not considered in User Space mode), HFI, IB, HPCE, and KMUX. 
        IMMED =<count>
            Number of immediate send slots per window required. Applies only to IBM Power7-IH processors. Default value is zero. 
        INSTANCES =<count>
            Specify number of network connections for each task on each network connection. The default instance count is 1. 
        IPV4
            Use Internet Protocol (IP) version 4 communications (default). 
        IPV6
            Use Internet Protocol (IP) version 6 communications. 
        LAPI
            Use the LAPI programming interface. 
        MPI
            Use the MPI programming interface. MPI is the default interface. 
        PAMI
            Use the PAMI programming interface. 
        SHMEM
            Use the OpenSHMEM programming interface. 
        SN_ALL
            Use all available switch networks (default). 
        SN_SINGLE
            Use one available switch network. 
        UPC
            Use the UPC programming interface. 
        US
            Use User Space communications. 
            Some examples of network specifications: 
        Instances=2,US,MPI,SN_ALL
            Create two user space connections for MPI communications on every switch network for each task. 
        US,MPI,Instances=3,Devtype=IB
            Create three user space connections for MPI communications on every InfiniBand network for each task. 
        IPV4,LAPI,SN_Single
            Create a IP version 4 connection for LAPI communications on one switch network for each task. 
        Instances=2,US,LAPI,MPI
            Create two user space connections each for LAPI and MPI communications on every switch network for each task. Note that SN_ALL is the default option so every switch network is used. Also note that Instances=2 specifies that two connections are established for each protocol (LAPI and MPI) and each task. If there are two networks and four tasks on the node then a total of 32 connections are established (2 instances x 2 protocols x 2 networks x 4 tasks). 

`
	sBatchNiceDesc            = `Run the job with an adjusted scheduling priority within Slurm. With no adjustment value the scheduling priority is decreased by 100. A negative nice value increases the priority, otherwise decreases it. The adjustment range is +/- 2147483645. Only privileged users can specify a negative adjustment. `
	sBatchNoRequeueDesc       = `Specifies that the batch job should never be requeued under any circumstances. Setting this option will prevent system administrators from being able to restart the job (for example, after a scheduled downtime), recover from a node failure, or be requeued upon preemption by a higher priority job. When a job is requeued, the batch script is initiated from its beginning. Also see the --requeue option. The JobRequeue configuration parameter controls the default behavior on the cluster. `
	sBatchNtasksPerCoreDesc   = `Request the maximum ntasks be invoked on each core. Meant to be used with the --ntasks option. Related to --ntasks-per-node except at the core level instead of the node level. NOTE: This option is not supported unless SelectType=cons_res is configured (either directly or indirectly on Cray systems) along with the node's core count. `
	sBatchNtasksPerNodeDesc   = `Request that ntasks be invoked on each node. If used with the --ntasks option, the --ntasks option will take precedence and the --ntasks-per-node will be treated as a maximum count of tasks per node. Meant to be used with the --nodes option. This is related to --cpus-per-task=ncpus, but does not require knowledge of the actual number of cpus on each node. In some cases, it is more convenient to be able to request that no more than a specific number of tasks be invoked on each node. Examples of this include submitting a hybrid MPI/OpenMP app where only one MPI "task/rank" should be assigned to each node while allowing the OpenMP portion to utilize all of the parallelism present in the node, or submitting a single setup/cleanup/monitoring job to each node of a pre-existing allocation as one step in a larger job script. `
	sBatchNtasksPerSocketDesc = `Request the maximum ntasks be invoked on each socket. Meant to be used with the --ntasks option. Related to --ntasks-per-node except at the socket level instead of the node level. NOTE: This option is not supported unless SelectType=cons_res is configured (either directly or indirectly on Cray systems) along with the node's socket count. `
	sBatchOvercommitDesc      = `Overcommit resources. When applied to job allocation, only one CPU is allocated to the job per node and options used to specify the number of tasks per node, socket, core, etc. are ignored. When applied to job step allocations (the srun command when executed within an existing job allocation), this option can be used to launch more than one task per CPU. Normally, srun will not allocate more than one process per CPU. By specifying --overcommit you are explicitly allowing more than one process per CPU. However no more than MAX_TASKS_PER_NODE tasks are permitted to execute per node. NOTE: MAX_TASKS_PER_NODE is defined in the file slurm.h and is not a variable, it is set at Slurm build time. `
	sBatchOutputDesc          = `Instruct Slurm to connect the batch script's standard output directly to the file name specified in the "filename pattern". By default both standard output and standard error are directed to the same file. For job arrays, the default file name is "slurm-%A_%a.out", "%A" is replaced by the job ID and "%a" with the array index. For other jobs, the default file name is "slurm-%j.out", where the "%j" is replaced by the job ID. See the filename pattern section below for filename specification options. `
	sBatchOpenModeDesc        = `    Open the output and error files using append or truncate mode as specified. The default value is specified by the system configuration parameter JobFileAppend.`
	sBatchParsableDesc        = `Outputs only the job id number and the cluster name if present. The values are separated by a semicolon. Errors will still be displayed. `
	sBatchPartitionDesc       = `Request a specific partition for the resource allocation. If not specified, the default behavior is to allow the slurm controller to select the default partition as designated by the system administrator. If the job can use more than one partition, specify their names in a comma separate list and the one offering earliest initiation will be used with no regard given to the partition name ordering (although higher priority partitions will be considered first). When the job is initiated, the name of the partition used will be placed first in the job record partition string. `
	sBatchPowerDesc           = `Comma separated list of power management plugin options. Currently available flags include: level (all nodes allocated to the job should have identical power caps, may be disabled by the Slurm configuration option PowerParameters=job_no_level). `
	sBatchPriorityDesc        = `Request a specific job priority. May be subject to configuration specific constraints. value should either be a numeric value or "TOP" (for highest possible value). Only Slurm operators and administrators can set the priority of a job. `
	sBatchProfileDesc         = `enables detailed data collection by the acct_gather_profile plugin. Detailed data are typically time-series that are stored in an HDF5 file for the job or an InfluxDB database depending on the configured plugin.

    All
        All data types are collected. (Cannot be combined with other values.)

    None
        No data types are collected. This is the default.
         (Cannot be combined with other values.)

    Energy
        Energy data is collected.

    Task
        Task (I/O, Memory, ...) data is collected.

    Lustre
        Lustre data is collected.

    Network
        Network (InfiniBand) data is collected. 

`
	sBatchPropagateDesc = `Allows users to specify which of the modifiable (soft) resource limits to propagate to the compute nodes and apply to their jobs. If no rlimit is specified, then all resource limits will be propagated. The following rlimit names are supported by Slurm (although some options may not be supported on some systems):

    ALL
        All limits listed below (default) 
    NONE
        No limits listed below 
    AS
        The maximum address space for a process 
    CORE
        The maximum size of core file 
    CPU
        The maximum amount of CPU time 
    DATA
        The maximum size of a process's data segment 
    FSIZE
        The maximum size of files created. Note that if the user sets FSIZE to less than the current size of the slurmd.log, job launches will fail with a 'File size limit exceeded' error. 
    MEMLOCK
        The maximum size that may be locked into memory 
    NOFILE
        The maximum number of open files 
    NPROC
        The maximum number of processes available 
    RSS
        The maximum resident set size 
    STACK
        The maximum stack size 

`
	sBatchQosDesc            = `Request a quality of service for the job. QOS values can be defined for each user/cluster/account association in the Slurm database. Users will be limited to their association's defined set of qos's when the Slurm configuration parameter, AccountingStorageEnforce, includes "qos" in its definition. `
	sBatchQuietDesc          = `Suppress informational messages from sbatch such as Job ID. Only errors will still be displayed. `
	sBatchRebootDesc         = `    Force the allocated nodes to reboot before starting the job. This is only supported with some system configurations and will otherwise be silently ignored. Only root, SlurmUser or admins can reboot nodes.`
	sBatchRequeueDesc        = `Specifies that the batch job should eligible to being requeue. The job may be requeued explicitly by a system administrator, after node failure, or upon preemption by a higher priority job. When a job is requeued, the batch script is initiated from its beginning. Also see the --no-requeue option. The JobRequeue configuration parameter controls the default behavior on the cluster. `
	sBatchReservationDesc    = `Allocate resources for the job from the named reservation. `
	sBatchOversubscribeDesc  = `The job allocation can over-subscribe resources with other running jobs. The resources to be over-subscribed can be nodes, sockets, cores, and/or hyperthreads depending upon configuration. The default over-subscribe behavior depends on system configuration and the partition's OverSubscribe option takes precedence over the job's option. This option may result in the allocation being granted sooner than if the --oversubscribe option was not set and allow higher system utilization, but application performance will likely suffer due to competition for resources. Also see the --exclusive option. `
	sBatchCoreSpecDesc       = `Count of specialized cores per node reserved by the job for system operations and not used by the application. The application will not use these cores, but will be charged for their allocation. Default value is dependent upon the node's configured CoreSpecCount value. If a value of zero is designated and the Slurm configuration option AllowSpecResourcesUsage is enabled, the job will be allowed to override CoreSpecCount and use the specialized resources on nodes it is allocated. This option can not be used with the --thread-spec option. `
	sBatchSignalDesc         = `When a job is within sig_time seconds of its end time, send it the signal sig_num. Due to the resolution of event handling by Slurm, the signal may be sent up to 60 seconds earlier than specified. sig_num may either be a signal number or name (e.g. "10" or "USR1"). sig_time must have an integer value between 0 and 65535. By default, no signal is sent before the job's end time. If a sig_num is specified without any sig_time, the default time will be 60 seconds. Use the "B:" option to signal only the batch shell, none of the other processes will be signaled. By default all job steps will be signaled, but not the batch shell itself. Use the "R:" option to allow this job to overlap with a reservation with MaxStartDelay set. To have the signal sent at preemption time see the preempt_send_user_signal SlurmctldParameter. `
	sBatchSocketsPerNodeDesc = `Restrict node selection to nodes with at least the specified number of sockets. See additional information under -B option above when task/affinity plugin is enabled. `
	sBatchSpreadJobDesc      = `Spread the job allocation over as many nodes as possible and attempt to evenly distribute tasks across the allocated nodes. This option disables the topology/tree plugin. `
	sBatchSwitchesDesc       = `When a tree topology is used, this defines the maximum count of switches desired for the job allocation and optionally the maximum time to wait for that number of switches. If Slurm finds an allocation containing more switches than the count specified, the job remains pending until it either finds an allocation with desired switch count or the time limit expires. It there is no switch count limit, there is no delay in starting the job. Acceptable time formats include "minutes", "minutes:seconds", "hours:minutes:seconds", "days-hours", "days-hours:minutes" and "days-hours:minutes:seconds". The job's maximum time delay may be limited by the system administrator using the SchedulerParameters configuration parameter with the max_switch_wait parameter option. On a dragonfly network the only switch count supported is 1 since communication performance will be highest when a job is allocate resources on one leaf switch or more than 2 leaf switches. The default max-time is the max_switch_wait SchedulerParameters. `
	sBatchTimeDesc           = `Set a limit on the total run time of the job allocation. If the requested time limit exceeds the partition's time limit, the job will be left in a PENDING state (possibly indefinitely). The default time limit is the partition's default time limit. When the time limit is reached, each task in each job step is sent SIGTERM followed by SIGKILL. The interval between signals is specified by the Slurm configuration parameter KillWait. The OverTimeLimit configuration parameter may permit the job to run longer than scheduled. Time resolution is one minute and second values are rounded up to the next minute.

A time limit of zero requests that no time limit be imposed. Acceptable time formats include "minutes", "minutes:seconds", "hours:minutes:seconds", "days-hours", "days-hours:minutes" and "days-hours:minutes:seconds". `
	sBatchTestOnlyDesc       = `Validate the batch script and return an estimate of when a job would be scheduled to run given the current job queue and all the other arguments specifying the job requirements. No job is actually submitted. `
	sBatchThreadSpecDesc     = `Count of specialized threads per node reserved by the job for system operations and not used by the application. The application will not use these threads, but will be charged for their allocation. This option can not be used with the --core-spec option. `
	sBatchThreadsPerCoreDesc = `Restrict node selection to nodes with at least the specified number of threads per core. NOTE: "Threads" refers to the number of processing units on each core rather than the number of application tasks to be launched per core. See additional information under -B option above when task/affinity plugin is enabled. `
	sBatchTimeMinDesc        = `Set a minimum time limit on the job allocation. If specified, the job may have its --time limit lowered to a value no lower than --time-min if doing so permits the job to begin execution earlier than otherwise possible. The job's time limit will not be changed after the job is allocated resources. This is performed by a backfill scheduling algorithm to allocate resources otherwise reserved for higher priority jobs. Acceptable time formats include "minutes", "minutes:seconds", "hours:minutes:seconds", "days-hours", "days-hours:minutes" and "days-hours:minutes:seconds". `
	sBatchTmpDesc            = `Specify a minimum amount of temporary disk space per node. Default units are megabytes unless the SchedulerParameters configuration parameter includes the "default_gbytes" option for gigabytes. Different units can be specified using the suffix [K|M|G|T]. `
	sBatchUidDesc            = `Attempt to submit and/or run a job as user instead of the invoking user id. The invoking user's credentials will be used to check access permissions for the target partition. User root may use this option to run jobs as a normal user in a RootOnly partition for example. If run as root, sbatch will drop its permissions to the uid specified after node allocation is successful. user may be the user name or numerical user ID. `
	sBatchUseMinNodesDesc    = `If a range of node counts is given, prefer the smaller count. `
	sBatchVersionDesc        = `Display version information and exit. `
	sBatchVerboseDesc        = `Increase the verbosity of sbatch's informational messages. Multiple -v's will further increase sbatch's verbosity. By default only errors will be displayed. `
	sBatchNodelistDesc       = `Request a specific list of hosts. The job will contain all of these hosts and possibly additional hosts as needed to satisfy resource requirements. The list may be specified as a comma-separated list of hosts, a range of hosts (host[1-5,7,...] for example), or a filename. The host list will be assumed to be a filename if it contains a "/" character. If you specify a minimum node or processor count larger than can be satisfied by the supplied host list, additional resources will be allocated on other nodes as needed. Duplicate node names in the list will be ignored. The order of the node names in the list is not important; the node names will be sorted by Slurm. `
	sBatchWaitDesc           = `Do not exit until the submitted job terminates. The exit code of the sbatch command will be the same as the exit code of the submitted job. If the job terminated due to a signal rather than a normal exit, the exit code will be set to 1. In the case of a job array, the exit code recorded will be the highest value for any task in the job array. `
	sBatchWaitAllNodesDesc   = `Controls when the execution of the command begins. By default the job will begin execution as soon as the allocation is made.

    0
        Begin execution as soon as allocation can be made. Do not wait for all nodes to be ready for use (i.e. booted). 
    1
        Do not begin execution until all nodes are ready for use. 

`
	sBatchWckeyDesc   = `Specify wckey to be used with job. If TrackWCKey=no (default) in the slurm.conf this value is ignored. `
	sBatchWrapDesc    = `Sbatch will wrap the specified command string in a simple "sh" shell script, and submit that script to the slurm controller. When --wrap is used, a script name and arguments may not be specified on the command line; instead the sbatch-generated wrapper script is used. `
	sBatchExcludeDesc = `Explicitly exclude certain nodes from the resources granted to the job. `
)

// List of support Slurm options
// map[string]struct{} enables querying supported options using:
// _, ok := sBatchSupportedArgs()["<option>"]
func sBatchSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"chdir":     struct{}{},
		"job-name":  struct{}{},
		"nodes":     struct{}{},
		"time":      struct{}{},
		"partition": struct{}{},
	}
}

// Slurm uses Short and Long command line options
// Save both with golang flag
type gnuFlag struct {
	Short string
	Long  string
	Value interface{}
}

// Use map to set command line options. map key is the same as Long option
type gnuFlags map[string]gnuFlag

// Check if either Long or Short flag is used
func lookupGnuArg(name string, spec gnuFlags) (string, error) {
	for k, v := range spec {
		// map key is the same as Long option
		if name == k || name == v.Short {
			return k, nil
		}
	}
	return "", errors.New("sbatch: unable to parse arguments")
}

func parseSBatchArgs(args []string) (gnuFlags, *flag.FlagSet, error) {

	flags := flag.NewFlagSet("sbatch", flag.ContinueOnError)

	options := make(gnuFlags)
	// Add each supported option for sbatch using golang flag and Short/Long options
	// TODO: update default values?
	jobArray := setFlagString(flags, "a", "array", "", sBatchArrayDesc)
	options["array"] = gnuFlag{
		Short: "a",
		Long:  "array",
		Value: jobArray,
	}
	chargeAccount := setFlagString(flags, "A", "account", "", sBatchAccountDesc)
	options["account"] = gnuFlag{
		Short: "A",
		Long:  "account",
		Value: chargeAccount,
	}
	jobAccounting := flags.String("acctg-freq", "", sBatchAcctgFreqDesc)
	options["acctg-freq"] = gnuFlag{
		Short: "",
		Long:  "acctg-freq",
		Value: jobAccounting,
	}
	nodeSelection := setFlagString(flags, "B", "extra-node-info", "", sBatchExtraNodeInfoDesc)
	options["extra-node-info"] = gnuFlag{
		Short: "B",
		Long:  "extra-node-info",
		Value: nodeSelection,
	}
	batch := flags.String("batch", "", sBatchBatchDesc)
	options["batch"] = gnuFlag{
		Short: "",
		Long:  "batch",
		Value: batch,
	}
	burstBuffer := flags.String("bb", "", sBatchBurstBufferDesc)
	options["bb"] = gnuFlag{
		Short: "",
		Long:  "bb",
		Value: burstBuffer,
	}
	burstBufferFile := flags.String("bbf", "", sBatchBurstBufferFileDesc)
	options["bbf"] = gnuFlag{
		Short: "",
		Long:  "bbf",
		Value: burstBufferFile,
	}
	beginTime := setFlagString(flags, "b", "begin", "", sBatchBeginDesc)
	options["begin"] = gnuFlag{
		Short: "b",
		Long:  "begin",
		Value: beginTime,
	}
	clusterConstraint := flags.String("cluster-constraint", "", sBatchClusterConstraintDesc)
	options["cluster-constraint"] = gnuFlag{
		Short: "",
		Long:  "cluster-constraint",
		Value: clusterConstraint,
	}
	comment := flags.String("comment", "", sBatchCommentDesc)
	options["comment"] = gnuFlag{
		Long:  "comment",
		Value: comment,
	}
	constraint := setFlagString(flags, "C", "constraint", "", sBatchConstraintDesc)
	options["constraint"] = gnuFlag{
		Short: "C",
		Long:  "constraint",
		Value: constraint,
	}
	contiguous := flags.Bool("contiguous", false, sBatchContiguousDesc)
	options["contiguous"] = gnuFlag{
		Short: "",
		Long:  "contiguous",
		Value: contiguous,
	}
	coresPerSocket := flags.Int("cores-per-socket", 1, sBatchCoresPerSocketDesc)
	options["cores-per-socket"] = gnuFlag{
		Short: "",
		Long:  "cores-per-socket",
		Value: coresPerSocket,
	}
	cpuFreq := flags.String("cpu-freq", "", sBatchCpuFreqDesc)
	options["cpu-freq"] = gnuFlag{
		Short: "",
		Long:  "cpu-freq",
		Value: cpuFreq,
	}
	cpusPerGpu := flags.Int("cpus-per-gpu", 0, sBatchCpusPerGpuDesc)
	options["cpus-per-gpu"] = gnuFlag{
		Short: "",
		Long:  "cpus-per-gpu",
		Value: cpusPerGpu,
	}
	cpusPerTask := setFlagInt(flags, "c", "cpus-per-task", 1, sBatchCpusPerTaskDesc)
	options["cpus-per-task"] = gnuFlag{
		Short: "c",
		Long:  "cpus-per-task",
		Value: cpusPerTask,
	}
	deadline := flags.String("deadline", "", sBatchDeadlineDesc)
	options["deadline"] = gnuFlag{
		Short: "",
		Long:  "deadline",
		Value: deadline,
	}
	delayBoot := flags.Int("delay-boot", 0, sBatchDelayBootDesc)
	options["delay-boot"] = gnuFlag{
		Short: "",
		Long:  "delay-boot",
		Value: delayBoot,
	}
	dependency := setFlagString(flags, "d", "dependency", "", sBatchDependencyDesc)
	options["dependency"] = gnuFlag{
		Short: "",
		Long:  "dependency",
		Value: dependency,
	}
	workingDirectory := setFlagString(flags, "D", "chdir", "", sBatchChdirDesc)
	options["chdir"] = gnuFlag{
		Short: "D",
		Long:  "chdir",
		Value: workingDirectory,
	}
	errorFile := setFlagString(flags, "e", "error", "", sBatchErrorDesc)
	options["error"] = gnuFlag{
		Short: "e",
		Long:  "error",
		Value: errorFile,
	}
	exclusive := flags.String("exclusive", "", sBatchExclusiveDesc)
	options["exclusive"] = gnuFlag{
		Long:  "exclusive",
		Value: exclusive,
	}
	copyEnvironment := flags.String("export", "NONE", sBatchExportDesc)
	options["export"] = gnuFlag{
		Long:  "export",
		Value: copyEnvironment,
	}
	exportFile := flags.String("export-file", "", sBatchExportFileDesc)
	options["export-file"] = gnuFlag{
		Long:  "export-file",
		Value: exportFile,
	}
	nodeFile := setFlagString(flags, "F", "nodefile", "", sBatchNodefileDesc)
	options["nodefile"] = gnuFlag{
		Short: "F",
		Long:  "nodefile",
		Value: nodeFile,
	}
	getUserEnv := flags.String("get-user-env", "", sBatchGetUserEnvDesc)
	options["get-user-env"] = gnuFlag{
		Long:  "get-user-env",
		Value: getUserEnv,
	}
	gid := flags.String("gid", "", sBatchGidDesc)
	options["gid"] = gnuFlag{
		Long:  "gid",
		Value: gid,
	}
	gpus := setFlagString(flags, "G", "gpus", "", sBatchGpusDesc)
	options["gpus"] = gnuFlag{
		Short: "G",
		Long:  "gpus",
		Value: gpus,
	}
	gpuBind := flags.String("gpu-bind", "", sBatchGpuBindDesc)
	options["gpu-bind"] = gnuFlag{
		Long:  "gpu-bind",
		Value: gpuBind,
	}
	gpuFreq := flags.String("gpu-freq", "", sBatchGpuFreqDesc)
	options["gpu-freq"] = gnuFlag{
		Long:  "gpu-freq",
		Value: gpuFreq,
	}
	gpusPerNode := flags.String("gpus-per-node", "", sBatchGpusPerNodeDesc)
	options["gpus-per-node"] = gnuFlag{
		Long:  "gpus-per-node",
		Value: gpusPerNode,
	}
	gpusPerSocket := flags.String("gpus-per-socket", "", sBatchGpusPerSocketDesc)
	options["gpus-per-socket"] = gnuFlag{
		Long:  "gpus-per-socket",
		Value: gpusPerSocket,
	}
	gpusPerTask := flags.String("gpus-per-task", "", sBatchGpusPerTaskDesc)
	options["gpus-per-task"] = gnuFlag{
		Long:  "gpus-per-task",
		Value: gpusPerTask,
	}
	genericResources := flags.String("gres", "", sBatchGresDesc)
	options["gres"] = gnuFlag{
		Long:  "gres",
		Value: genericResources,
	}
	gresFlags := flags.String("gres-flags", "", sBatchGresFlagsDesc)
	options["gres-flags"] = gnuFlag{
		Long:  "gres-flags",
		Value: gresFlags,
	}
	hold := setFlagBool(flags, "H", "hold", false, sBatchHoldDesc)
	options["hold"] = gnuFlag{
		Long:  "hold",
		Value: hold,
	}
	hint := flags.String("hint", "", sBatchHintDesc)
	options["hint"] = gnuFlag{
		Long:  "hint",
		Value: hint,
	}
	ignorePbs := flags.Bool("ignore-pbs", true, sBatchIgnorePbsDesc)
	options["ignore-pbs"] = gnuFlag{
		Long:  "ignore-pbs",
		Value: ignorePbs,
	}
	input := setFlagString(flags, "i", "input", "", sBatchInputDesc)
	options["input"] = gnuFlag{
		Long:  "input",
		Value: input,
	}
	jobName := setFlagString(flags, "J", "job-name", "", sBatchJobNameDesc)
	options["job-name"] = gnuFlag{
		Short: "J",
		Long:  "job-name",
		Value: jobName,
	}
	noKill := setFlagString(flags, "k", "no-kill", "", sBatchNoKillDesc)
	options["no-kill"] = gnuFlag{
		Short: "k",
		Long:  "no-kill",
		Value: noKill,
	}
	killOnInvalidDep := flags.Bool("kill-on-invalid-dep", false, sBatchKillOnInvalidDepDesc)
	options["kill-on-invalid-dep"] = gnuFlag{
		Long:  "kill-on-invalid-dep",
		Value: killOnInvalidDep,
	}
	licenses := setFlagString(flags, "L", "licenses", "", sBatchLicensesDesc)
	options["licenses"] = gnuFlag{
		Short: "L",
		Long:  "licenses",
		Value: licenses,
	}
	clusters := setFlagString(flags, "M", "clusters", "", sBatchClustersDesc)
	options["clusters"] = gnuFlag{
		Long:  "clusters",
		Value: clusters,
	}
	distribution := setFlagString(flags, "m", "distribution", "", sBatchDistributionDesc)
	options["description"] = gnuFlag{
		Long:  "description",
		Value: distribution,
	}
	eventNotification := flags.String("mail-type", "", sBatchMailTypeDesc)
	options["mail-type"] = gnuFlag{
		Long:  "mail-type",
		Value: eventNotification,
	}
	emailAddress := flags.String("mail-user", "", sBatchMailUserDesc)
	options["mail-user"] = gnuFlag{
		Long:  "mail-user",
		Value: emailAddress,
	}
	mcsLabel := flags.String("mcs-label", "", sBatchMcsLabelDesc)
	options["mcs-label"] = gnuFlag{
		Long:  "mcs-label",
		Value: mcsLabel,
	}
	memory := flags.String("mem", "", sBatchMemDesc)
	options["mem"] = gnuFlag{
		Long:  "mem",
		Value: memory,
	}
	memPerCpu := flags.String("mem-per-cpu", "", sBatchMemPerCpuDesc)
	options["mem-per-cpu"] = gnuFlag{
		Long:  "mem-per-cpu",
		Value: memPerCpu,
	}
	memPerGpu := flags.String("mem-per-gpu", "", sBatchMemPerGpuDesc)
	options["mem-per-gpu"] = gnuFlag{
		Long:  "mem-per-gpu",
		Value: memPerGpu,
	}
	memBind := flags.String("mem-bind", "", sBatchMemBindDesc)
	options["mem-bind"] = gnuFlag{
		Long:  "mem-bind",
		Value: memBind,
	}
	mincpus := flags.Int("mincpus", 1, sBatchMincpusDesc)
	options["mincpus"] = gnuFlag{
		Long:  "mincpus",
		Value: mincpus,
	}
	nodeCount := setFlagInt(flags, "N", "nodes", 1, sBatchNodesDesc)
	options["nodes"] = gnuFlag{
		Short: "N",
		Long:  "nodes",
		Value: nodeCount,
	}
	cpuCount := setFlagInt(flags, "n", "ntasks", 1, sBatchNtasksDesc)
	options["ntasks"] = gnuFlag{
		Short: "n",
		Long:  "ntasks",
		Value: cpuCount,
	}
	network := flags.String("network", "", sBatchNetworkDesc)
	options["network"] = gnuFlag{
		Long:  "network",
		Value: network,
	}
	nice := flags.String("nice", "", sBatchNiceDesc)
	options["nice"] = gnuFlag{
		Long:  "nice",
		Value: nice,
	}
	noRequeue := flags.Bool("no-requeue", false, sBatchNoRequeueDesc)
	options["no-requeue"] = gnuFlag{
		Long:  "no-requeue",
		Value: noRequeue,
	}
	ntasksPerCore := flags.Int("ntasks-per-core", 1, sBatchNtasksPerCoreDesc)
	options["ntasks-per-core"] = gnuFlag{
		Long:  "ntasks-per-core",
		Value: ntasksPerCore,
	}
	tasksPerNode := flags.Int("ntasks-per-node", 1, sBatchNtasksPerNodeDesc)
	options["ntasks-per-node"] = gnuFlag{
		Long:  "ntasks-per-node",
		Value: tasksPerNode,
	}
	ntasksPerSocket := flags.Int("ntasks-per-socket", 1, sBatchNtasksPerSocketDesc)
	options["ntasks-per-socket"] = gnuFlag{
		Long:  "ntasks-per-socket",
		Value: ntasksPerSocket,
	}
	overcommit := setFlagString(flags, "O", "overcommit", "", sBatchOvercommitDesc)
	options["overcommit"] = gnuFlag{
		Long:  "overcommit",
		Value: overcommit,
	}
	outputFile := setFlagString(flags, "o", "output", "", sBatchOutputDesc)
	options["output"] = gnuFlag{
		Short: "o",
		Long:  "output",
		Value: outputFile,
	}
	openMode := flags.String("open-mode", "", sBatchOpenModeDesc)
	options["open-mode"] = gnuFlag{
		Long:  "open-mode",
		Value: openMode,
	}
	parsable := flags.Bool("parsable", false, sBatchParsableDesc)
	options["parsable"] = gnuFlag{
		Long:  "parsable",
		Value: parsable,
	}
	queue := setFlagString(flags, "p", "partition", "default", sBatchPartitionDesc)
	options["partition"] = gnuFlag{
		Short: "p",
		Long:  "partition",
		Value: queue,
	}
	power := flags.String("power", "", sBatchPowerDesc)
	options["power"] = gnuFlag{
		Long:  "power",
		Value: power,
	}
	priority := flags.String("priority", "", sBatchPriorityDesc)
	options["priority"] = gnuFlag{
		Long:  "priority",
		Value: priority,
	}
	profile := flags.String("profile", "", sBatchProfileDesc)
	options["profile"] = gnuFlag{
		Long:  "profile",
		Value: profile,
	}
	propagate := flags.String("propagate", "", sBatchPropagateDesc)
	options["propagate"] = gnuFlag{
		Long:  "propagate",
		Value: propagate,
	}
	qos := setFlagString(flags, "q", "qos", "", sBatchQosDesc)
	options["qos"] = gnuFlag{
		Long:  "qos",
		Value: qos,
	}
	quiet := setFlagBool(flags, "Q", "quiet", false, sBatchQuietDesc)
	options["quiet"] = gnuFlag{
		Long:  "quiet",
		Value: quiet,
	}
	reboot := flags.Bool("reboot", false, sBatchRebootDesc)
	options["reboot"] = gnuFlag{
		Long:  "reboot",
		Value: reboot,
	}
	jobRestart := flags.Bool("requeue", false, sBatchRequeueDesc)
	options["requeue"] = gnuFlag{
		Long:  "requeue",
		Value: jobRestart,
	}
	reservation := flags.String("reservation", "", sBatchReservationDesc)
	options["reservation"] = gnuFlag{
		Long:  "reservation",
		Value: reservation,
	}
	oversubscribe := setFlagBool(flags, "s", "oversubscribe", false, sBatchOversubscribeDesc)
	options["oversubscribe"] = gnuFlag{
		Long:  "oversubscribe",
		Value: oversubscribe,
	}
	coreSpec := setFlagInt(flags, "S", "core-spec", 1, sBatchCoreSpecDesc)
	options["core-spec"] = gnuFlag{
		Long:  "core-spec",
		Value: coreSpec,
	}
	signal := flags.String("signal", "", sBatchSignalDesc)
	options["signal"] = gnuFlag{
		Long:  "signal",
		Value: signal,
	}
	socketPerNode := flags.Int("sockets-per-node", 1, sBatchSocketsPerNodeDesc)
	options["sockets-per-node"] = gnuFlag{
		Long:  "sockets-per-node",
		Value: socketPerNode,
	}
	spreadJob := flags.Bool("spread-job", false, sBatchSpreadJobDesc)
	options["spread-job"] = gnuFlag{
		Long:  "spread-job",
		Value: spreadJob,
	}
	switches := flags.String("switches", "", sBatchSwitchesDesc)
	options["switches"] = gnuFlag{
		Long:  "switches",
		Value: switches,
	}
	time := setFlagString(flags, "t", "time", "", sBatchTimeDesc)
	options["time"] = gnuFlag{
		Long:  "time",
		Value: time,
	}
	testOnly := flags.Bool("test-only", false, sBatchTestOnlyDesc)
	options["test-only"] = gnuFlag{
		Long:  "test-only",
		Value: testOnly,
	}
	threadSpec := flags.Int("thread-spec", 1, sBatchThreadSpecDesc)
	options["thread-spec"] = gnuFlag{
		Long:  "thread-spec",
		Value: threadSpec,
	}
	threadsPerCore := flags.Int("threads-per-core", 1, sBatchThreadsPerCoreDesc)
	options["threads-per-core"] = gnuFlag{
		Long:  "threads-per-core",
		Value: threadsPerCore,
	}
	timeMin := flags.String("time-min", "", sBatchTimeMinDesc)
	options["time-min"] = gnuFlag{
		Long:  "time-min",
		Value: timeMin,
	}
	tmp := flags.String("tmp", "", sBatchTmpDesc)
	options["tmp"] = gnuFlag{
		Long:  "tmp",
		Value: tmp,
	}
	uid := flags.String("uid", "", sBatchUidDesc)
	options["uid"] = gnuFlag{
		Long:  "uid",
		Value: uid,
	}
	useMinNodes := flags.Bool("use-min-nodes", true, sBatchUseMinNodesDesc)
	options["use-min-nodes"] = gnuFlag{
		Long:  "use-min-nodes",
		Value: useMinNodes,
	}
	version := setFlagBool(flags, "V", "version", false, sBatchVersionDesc)
	options["version"] = gnuFlag{
		Long:  "version",
		Value: version,
	}
	verbose := setFlagBool(flags, "v", "verbose", false, sBatchVerboseDesc)
	options["verbose"] = gnuFlag{
		Long:  "verbose",
		Value: verbose,
	}
	nodeList := setFlagString(flags, "w", "nodelist", "", sBatchNodelistDesc)
	options["nodelist"] = gnuFlag{
		Long:  "nodelist",
		Value: nodeList,
	}
	wait := setFlagBool(flags, "W", "wait", false, sBatchWaitDesc)
	options["wait"] = gnuFlag{
		Long:  "wait",
		Value: wait,
	}
	waitAllNodes := flags.Int("wait-all-nodes", 0, sBatchWaitAllNodesDesc)
	options["wait-all-nodes"] = gnuFlag{
		Long:  "wait-all-nodes",
		Value: waitAllNodes,
	}
	jobProject := flags.String("wckey", "", sBatchWckeyDesc)
	options["wckey"] = gnuFlag{
		Long:  "wckey",
		Value: jobProject,
	}
	wrap := flags.String("wrap", "", sBatchWrapDesc)
	options["wrap"] = gnuFlag{
		Long:  "wrap",
		Value: wrap,
	}
	exclude := setFlagString(flags, "x", "exclude", "", sBatchExcludeDesc)
	options["exclude"] = gnuFlag{
		Long:  "exclude",
		Value: exclude,
	}

	if flags.Parse(false, args) != nil {
		return nil, &flag.FlagSet{}, errors.New("sbatch: cannot process flags")
	}

	return options, flags, nil
}

const (
	sCancelAccountDesc     = `Restrict the scancel operation to jobs under this charge account.`
	sCancelBatchDesc       = `By default, signals other than SIGKILL are not sent to the batch step (the shell script). With this option scancel signals only the batch step, but not any other steps. This is useful when the shell script has to trap the signal and take some application defined action. Note that most shells cannot handle signals while a command is running (child process of the batch step), the shell use to wait wait until the command ends to then handle the signal. Children of the batch step are not signaled with this option, use -f, --full instead. NOTE: If used with -f, --full, this option ignored. NOTE: This option is not applicable if step_id is specified. NOTE: The shell itself may exit upon receipt of many signals. You may avoid this by explicitly trap signals within the shell script (e.g. "trap <arg> <signals>"). See the shell documentation for details.`
	sCancelCtldDesc        = `Send the job signal request to the slurmctld daemon rather than directly to the slurmd daemons. This increases overhead, but offers better fault tolerance. This is the default behavior on architectures using front end nodes (e.g. Cray ALPS computers) or when the --clusters option is used. `
	sCancelFullDesc        = `By default, signals other than SIGKILL are not sent to the batch step (the shell script). With this option scancel signals also the batch script and its children processes. Most shells cannot handle signals while a command is running (child process of the batch step), the shell use to wait until the command ends to then handle the signal. Unlike -b, --batch, children of the batch step are also signaled with this option. NOTE: srun steps are also children of the batch step, so steps are also signaled with this option.`
	sCancelHurryDesc       = `Do not stage out any burst buffer data.`
	sCancelInteractiveDesc = `Interactive mode. Confirm each job_id.step_id before performing the cancel operation.`
	sCancelClustersDesc    = `Cluster to issue commands to. Implies --ctld. Note that the SlurmDBD must be up for this option to work properly.`
	sCancelJobnameDesc     = `Restrict the scancel operation to jobs with this job name.`
	sCancelPartitionDesc   = `Restrict the scancel operation to jobs in this partition.`
	sCancelQosDesc         = `Restrict the scancel operation to jobs with this quality of service.`
	sCancelQuietDesc       = `Do not report an error if the specified job is already completed. This option is incompatible with the --verbose option.`
	sCancelReservationDesc = `Restrict the scancel operation to jobs with this reservation name.`
	sCancelSiblingDesc     = `Remove an active sibling job from a federated job.`
	sCancelSignalDesc      = `The name or number of the signal to send. If this option is not used the specified job or step will be terminated. Note. If this option is used the signal is sent directly to the slurmd where the job is running bypassing the slurmctld thus the job state will not change even if the signal is delivered to it. Use the scontrol command if you want the job state change be known to slurmctld.`
	sCancelStateDesc       = `Restrict the scancel operation to jobs in this state. job_state_name may have a value of either "PENDING", "RUNNING" or "SUSPENDED".`
	sCancelUserDesc        = `Restrict the scancel operation to jobs owned by this user.`
	sCancelVerboseDesc     = `Print additional logging. Multiple v's increase logging detail. This option is incompatible with the --quiet option.`
	sCancelVersionDesc     = `Print the version number of the scancel command.`
	sCancelNodelistDesc    = `Cancel any jobs using any of the given hosts. The list may be specified as a comma-separated list of hosts, a range of hosts (host[1-5,7,...] for example), or a filename. The host list will be assumed to be a filename only if it contains a "/" character.`
	sCancelWckeyDesc       = `Restrict the scancel operation to jobs using this workload characterization key.`
)

func sCancelSupportedArgs() map[string]struct{} {
	return map[string]struct{}{
		"partition": struct{}{},
	}
}

func parseSCancelArgs(args []string) (gnuFlags, *flag.FlagSet, error) {

	flags := flag.NewFlagSet("scancel", flag.ContinueOnError)

	options := make(gnuFlags)
	// Add supported options for scancel
	account := setFlagString(flags, "A", "account", "", sCancelAccountDesc)
	options["account"] = gnuFlag{
		Short: "A",
		Long:  "account",
		Value: account,
	}
	batch := setFlagBool(flags, "b", "batch", false, sCancelBatchDesc)
	options["batch"] = gnuFlag{
		Short: "b",
		Long:  "batch",
		Value: batch,
	}
	ctld := flags.Bool("ctld", false, sCancelCtldDesc)
	options["ctld"] = gnuFlag{
		Long:  "ctld",
		Value: ctld,
	}
	full := flags.Bool("full", false, sCancelFullDesc)
	options["full"] = gnuFlag{
		Long:  "full",
		Value: full,
	}
	hurry := setFlagBool(flags, "H", "hurry", false, sCancelHurryDesc)
	options["hurry"] = gnuFlag{
		Short: "H",
		Long:  "hurry",
		Value: hurry,
	}
	interactive := setFlagBool(flags, "i", "interactive", false, sCancelInteractiveDesc)
	options["interactive"] = gnuFlag{
		Short: "i",
		Long:  "interactive",
		Value: interactive,
	}
	clusters := setFlagString(flags, "M", "clusters", "", sCancelClustersDesc)
	options["clusters"] = gnuFlag{
		Short: "M",
		Long:  "clusters",
		Value: clusters,
	}
	jobname := setFlagString(flags, "n", "jobname", "", sCancelJobnameDesc)
	options["jobname"] = gnuFlag{
		Short: "n",
		Long:  "jobname",
		Value: jobname,
	}
	partition := setFlagString(flags, "p", "partition", "", sCancelPartitionDesc)
	options["partition"] = gnuFlag{
		Short: "p",
		Long:  "partition",
		Value: partition,
	}
	qos := setFlagString(flags, "q", "qos", "", sCancelQosDesc)
	options["qos"] = gnuFlag{
		Short: "q",
		Long:  "qos",
		Value: qos,
	}
	quiet := setFlagBool(flags, "Q", "quiet", false, sCancelQuietDesc)
	options["quiet"] = gnuFlag{
		Short: "Q",
		Long:  "quiet",
		Value: quiet,
	}
	reservation := setFlagString(flags, "R", "reservation", "", sCancelReservationDesc)
	options["reservations"] = gnuFlag{
		Short: "R",
		Long:  "reservation",
		Value: reservation,
	}
	sibling := flags.String("sibling", "", sCancelSiblingDesc)
	options["sibling"] = gnuFlag{
		Long:  "sibling",
		Value: sibling,
	}
	signal := setFlagString(flags, "s", "signal", "", sCancelSignalDesc)
	options["signal"] = gnuFlag{
		Short: "s",
		Long:  "signal",
		Value: signal,
	}
	state := setFlagString(flags, "t", "state", "", sCancelStateDesc)
	options["state"] = gnuFlag{
		Short: "t",
		Long:  "state",
		Value: state,
	}
	user := setFlagString(flags, "u", "user", "", sCancelUserDesc)
	options["user"] = gnuFlag{
		Short: "u",
		Long:  "user",
		Value: user,
	}
	verbose := setFlagBool(flags, "v", "verbose", false, sCancelVerboseDesc)
	options["verbose"] = gnuFlag{
		Short: "v",
		Long:  "verbose",
		Value: verbose,
	}
	version := setFlagBool(flags, "V", "version", false, sCancelVersionDesc)
	options["version"] = gnuFlag{
		Short: "V",
		Long:  "version",
		Value: version,
	}
	nodelist := setFlagString(flags, "w", "nodelist", "", sCancelNodelistDesc)
	options["nodelist"] = gnuFlag{
		Short: "w",
		Long:  "nodelist",
		Value: nodelist,
	}
	wckey := flags.String("wckey", "", sCancelWckeyDesc)
	options["wckey"] = gnuFlag{
		Long:  "wckey",
		Value: wckey,
	}

	if flags.Parse(false, args) != nil {
		return nil, &flag.FlagSet{}, errors.New("scancel: connot process flags")
	}

	return options, flags, nil
}

// TODO: add parsing functions for squeue
