package sge

import (
	"errors"
	"regexp"
	"strconv"
)

const (
	mcCoresName = "mc_cores"
	mcRamName   = "mc_ram"
	mcGpusName  = "mc_gpus"
)

func decodeGpusReq(req string) (gpus int, err error) {
	re := regexp.MustCompile("^[a-zA-z0-9]+:")
	te := regexp.MustCompile("[0-9]+$")
	if match := te.FindString(req); len(match) > 0 {
		if numGpus, perr := strconv.ParseInt(match, 10, 64); perr == nil {
			if gpuType := re.FindString(req); len(gpuType) > 0 {
				err = errors.New("GPU type not supported")
				return
			}
			gpus = int(numGpus)
			return
		}
	}
	err = errors.New("Invalid gpu request")
	return
}
