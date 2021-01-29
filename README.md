# jarvice-hpc

JARVICE-HPC contains plugin clients to manage jobs on the [JARVICE](https://www.nimbix.net/platform) platform with traditional HPC schedulers via the [JARVICE API](https://jarvice.readthedocs.io/en/latest/api/).

Available plugins:
* Sun Grid Engine (SGE)
* Slurm

## Prerequisites

### Supported platform

JARVICE-HPC targets the Linux OS and has currently been tested with:

* Ubuntu Bionic
* CentOS 8

### Building requirements

JARVICE-HPC is built using GoLang 1.14 Docker container

* Install Docker: https://docs.docker.com/engine/install/

## Install

### SGE

```
git clone https://github.com/nimbix/jarvice-hpc
cd jarvice-hpc
./install.sh sge
```

### Slurm

```
git clone https://github.com/nimbix/jarvice-hpc
cd jarvice-hpc
./install.sh slurm
```

## Running SGE jobs

### Configure JARVICE credentials

The user credentials to use with the JARVICE API need to be configured before submitting work with a JARVICE-HPC plugin. The following command will write a configuration file to: ${HOME}/.config/jarvice-hpc

```
jarvice login -cluster <cluster-name> \
    -username <jarvice-username> \
    -apikey <jarvice-apikey> \
    -endpoint <jarvice-api-url> \
    -vault <jarvice-vault>
```
* `cluster-name: user provided name to label cluster in configuartion file (e.g. default)`
* `jarvice-username: username on target JARVICE platform`
* `jarvice-apikey: apikey on target JARVICE platform`
*Note* [Find JARVICE username and API key](https://support.nimbix.net/hc/en-us/articles/209770783-Where-do-I-find-my-JARVICE-API-Key-)
* `jarvice-api-url: endpoint for JARVICE API (e.g. https://api.jarvice.com/)`
* `jarvice-vault: JARVICE vault to use with HPC jobs (e.g. drop)`
*Note* Find available vaults [here](https://vaults.jarvice.com) with JARVICE username and apikey

The cluster configured by `jarvice login` will be used by all JARVICE-HPC plugin commands. `ephemeral` vaults are currently not supported

### Simple SGE job

examples/sgescript:
```
#!/bin/bash
#$ -N serial job test    # Job name
pwd; hostname; date
echo 'Hello World'
cat /etc/issue
sleep 30
echo 'Exiting'
```

This job script will be submitted to the JARVICE platform configured by the user using the public JARVICE API. The first several lines set the jobs shell and SGE options using the '$' directive. To submit this job to a queue:

1) List available queues

```
qconf
```

Example output
```
large
med
small
```

2) Submit job script to desired queue

```
qsub -q <queue-name> examples/sgescript
```

Example output
```
/home/khill
jarvice-job-7885-b5bbw
Wed Jan 20 19:04:08 UTC 2021
Hello World
Ubuntu 16.04.5 LTS \n \l

Exiting
```

*NOTE* Flags set on the command line will override options set inside a jobscript


### Muli Node SGE job

examples/sgemulti:
```
#!/bin/bash
#$ -N hpc job test    # Job name
pwd; hostname; date
echo 'Hello World'
/usr/local/JARVICE/tools/bin/python_ssh_test 60
mpirun --hostfile /etc/JARVICE/nodes -pernode hostname
sleep 30
echo 'Exiting'
```

Submit job script with multiple nodes

```
qsub -q <queue-name> -pe hpc <number-nodes> examples/sgemulti
```

Example output
```
jarvice-job-7859-clv5h
Tue Jan 19 19:24:12 UTC 2021
Hello World
Parallel slaves ready in 27 second(s)
jarvice-job-7859-clv5h
jarvice-job-7859-rltk8
Exiting
```

## Running Slum jobs

[See Configure JARVICE credentials](#configure-jarvice-credentials)

### Simple Slurm job

examples/sgescript:
```
#!/bin/bash
#SBATCH --job-name=serial job test    # Job name
pwd; hostname; date
echo 'Hello World'
cat /etc/issue
sleep 30
echo 'Exiting'
```

This job script will be submitted to the JARVICE platform configured by the user using the public JARVICE API. The first several lines set the jobs shell and Slurm options using the 'SBATCH' directive. To submit this job to a queue:

1) List available partitions

```
sinfo
```

Example output
```
PARTITION	AVAIL	TIMELIMIT	NODES	STATE	NODELIST
large		up	    infinite	4	    idle	n3[0-3]
med	        up	    infinite	2	    idle	n0[0-1]
small		up	    infinite	1	    idle	n0[0-0]
```

2) Submit job script to desired partition

```
sbatch -p <partition-name> examples/slurmscript
```

Example output
```
/home/khill
jarvice-job-7885-b5bbw
Wed Jan 20 19:04:08 UTC 2021
Hello World
Ubuntu 16.04.5 LTS \n \l

Exiting
```

*NOTE* Flags set on the command line will override options set inside a jobscript


### Muli Node Slurm job

examples/slurmmulti:
```
#!/bin/bash
#SBATCH --job-name hpc job test    # Job name
pwd; hostname; date
echo 'Hello World'
/usr/local/JARVICE/tools/bin/python_ssh_test 60
mpirun --hostfile /etc/JARVICE/nodes -pernode hostname
sleep 30
echo 'Exiting'
```

Submit job script with multiple nodes

```
sbatch -p <partition-name> -N <number-nodes> examples/slurmmulti
```

Example output
```
jarvice-job-7859-clv5h
Tue Jan 19 19:24:12 UTC 2021
Hello World
Parallel slaves ready in 27 second(s)
jarvice-job-7859-clv5h
jarvice-job-7859-rltk8
Exiting
```
## Authors

* **Kenneth Hill** - *Initial work* - ken.hill@nimbix.net

## License

This project uses an Open Source license - see the [LICENSE](LICENSE) file for details

