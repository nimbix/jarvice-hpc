# jarvice-hpc

JARVICE-HPC contains plugin clients to manage jobs on the [JARVICE](https://www.nimbix.net/platform) platform with traditional HPC schedulers via the [JARVICE API](https://jarvice.readthedocs.io/en/latest/api/).

Available plugins:
* Sun Grid Engine (SGE)
* Slurm

## Prerequisites

### Supported platform

JARVICE-HPC targets the Linux OS and has currently been tested on:

* Ubuntu Bionic
* CentOS 8

### Building requirements

JARVICE-HPC is built using GoLang 1.14

* Build using Docker: https://docs.docker.com/engine/install/

## Installing

### Building JARVICE-HPC

```
docker run -ti --rm -v "$PWD":/usr/src/jarvice-hpc \
    -w /usr/src/jarvice-hpc \
    -e GOOS=darwin golang:1.14 \
    /bin/bash -c "mkdir -p /go/src/jarvice.io \
    && ln -s /usr/src/jarvice-hpc /go/src/jarvice.io \
    && go build -v -o jarvice <sge|slurm>.go"
```

### Installing SGE plugin for JARVICE-HPC

```
INSTALL_PREFIX=/usr/local/bin
mv jarvice ${INSTALL_PREFIX}/jarvice
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/qstat
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/qconf
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/qsub
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/qacct
```

### Installing Slurm plugin for JARVICE-HPC

```
INSTALL_PREFIX=/usr/local/bin
mv jarvice ${INSTALL_PREFIX}/jarvice
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/sbatch
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/scancel
ln -s ${INSTALL_PREFIX}/jarvice ${INSTALL_PREFIX}/squeue
```

## Running jobs

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

### Simple SGE job

sgejobscript:
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
qsub -l mc_cluster=default -q <queue-name> sgejobscript
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

### Muli Node SGE job

sgehpcjobscript:
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
qsub -l mc_cluster=default -q <queue-name> -pe hpc <number-nodes> sgehpcjobscript
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

