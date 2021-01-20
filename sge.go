package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	core "jarvice.io/jarvice-hpc/core"
	sge "jarvice.io/jarvice-hpc/sge"
)

func main() {
	sge.SgeCliDebug()
	switch command := filepath.Base(os.Args[0]); command {
	case "jarvice":
		if err := core.Jarvice(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "qstat":
		if err := sge.QStat(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "qconf":
		if err := sge.QConf(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "qsub":
		if err := sge.QSub(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	case "qacct":
		if err := sge.QAcct(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	default:
		fmt.Println("Unknown Command")
	}

}
