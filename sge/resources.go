package sge

import (
	"errors"
	"math"
	"regexp"
	"strconv"
)

const (
	mcCoresName = "mc_cores"
	mcRamName   = "mc_ram"
	mcGpusName  = "mc_gpus"
)

func decodeMemReq(req string) (mem int, err error) {
	re := regexp.MustCompile("^[0-9]+")
	te := regexp.MustCompile("[KMGT]$")
	if match := re.FindString(req); len(match) > 0 {
		if base, perr := strconv.ParseInt(match, 10, 64); perr == nil {
			if mag := te.FindString(req); len(mag) > 0 {
				switch mag {
				case "K":
					mem = int(base) * 1024
				case "M":
					mem = int(base) * 1024 * 1024
				case "G":
					mem = int(base) * 1024 * 1024 * 1024
				case "T":
					mem = int(base) * 1024 * 1024 * 1024 * 1024
				}

			} else {
				mem = int(base) * 1024 * 1024
			}
			mem = int(math.Ceil(float64(mem) / float64((1024 * 1024 * 1024))))
			return
		}
	}
	err = errors.New("Invalid mem request")
	return
}

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
