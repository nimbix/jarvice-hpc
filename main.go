package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	core "jarvice.io/jarvice-hpc/core"
	slurm "jarvice.io/jarvice-hpc/slurm"
)

func main() {
	switch command := filepath.Base(os.Args[0]); command {
	case "jarvice":
		if err := core.Jarvice(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "sbatch":
		if err := slurm.SBatch(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "scancel":
		if err := slurm.SCancel(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "squeue":
		if err := slurm.SQueue(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	default:
		fmt.Println("Unknown Command")
	}

}
