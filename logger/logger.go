package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	LOG_ENABLE               = "JARVICE_HPC_LOGLEVEL"
	LOG_PATH                 = "JARVICE_HPC_LOGPATH"
	LOG_TIMEOUT              = "JARVICE_HPC_TIMEOUT"
	LOG_DEFAULT_TIMEOUT      = 24
	JARVICE_DEBUG_LOGGING    = 10
	JARVICE_INFO_LOGGING     = 20
	JARVICE_WARNING_LOGGING  = 30
	JARVICE_ERROR_LOGGING    = 40
	JARVICE_CRITICAL_LOGGING = 50
)

var (
	Log *log.Logger
)

func init() {
	logPath := "/tmp/"
	if env := os.Getenv(LOG_PATH); len(env) > 0 {
		logPath = env
	}
	timeout := LOG_DEFAULT_TIMEOUT
	if env := os.Getenv(LOG_TIMEOUT); len(env) > 0 {
		if t, err := strconv.Atoi(env); err == nil {
			timeout = t
		}
	}
	logfile := logPath + "jarvice-hpc.log"
	if f, err := os.Open(logfile); err == nil {
		scanner := bufio.NewScanner(f)
		scanner.Scan()
		f.Close()
		if tag, terr := time.Parse(time.RFC3339, scanner.Text()); terr == nil {
			if int(time.Since(tag).Hours()) > timeout {
				os.Remove(logfile)
			}
		} else {
			os.Remove(logfile)
		}
	}
	f, err := os.OpenFile(logfile,
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("logger cannot open file: %v",
			fmt.Errorf("LogWriter: OpenFile: %w", err))
	}
	if stat, serr := f.Stat(); serr == nil {
		if stat.Size() == 0 {
			f.WriteString(time.Now().Format(time.RFC3339) + "\n")
			f.Sync()
		}
	}
	wrt := io.MultiWriter(os.Stderr, f)
	Log = log.New(wrt, "", log.LstdFlags)
}

func LogLevel() int {
	if env, err := strconv.Atoi(os.Getenv(LOG_ENABLE)); err == nil {
		return env
	} else {
		return JARVICE_CRITICAL_LOGGING
	}
}

func getLogLevel(level int) string {
	switch level := level; level {
	case JARVICE_DEBUG_LOGGING:
		return "DEBUG"
	case JARVICE_INFO_LOGGING:
		return "INFO"
	case JARVICE_WARNING_LOGGING:
		return "WARNING"
	case JARVICE_ERROR_LOGGING:
		return "ERROR"
	default:
		return "CRITICAL"
	}
}

func DebugObj(name string, v interface{}) {
	level := JARVICE_DEBUG_LOGGING
	if LogLevel() <= level {
		data, _ := json.MarshalIndent(v, "", " ")
		Log.Printf("%s %s:\n%s\n", getLogLevel(level), name, data)
	}
}

func DebugPrintf(format string, a ...interface{}) {
	level := JARVICE_DEBUG_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func InfoObj(name string, v interface{}) {
	level := JARVICE_INFO_LOGGING
	if LogLevel() <= level {
		data, _ := json.MarshalIndent(v, "", " ")
		Log.Printf("%s %s:\n%s\n", getLogLevel(level), name, data)
	}
}

func InfoPrintf(format string, a ...interface{}) {
	level := JARVICE_INFO_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func WarningObj(name string, v interface{}) {
	level := JARVICE_WARNING_LOGGING
	if LogLevel() <= level {
		data, _ := json.MarshalIndent(v, "", " ")
		Log.Printf("%s %s:\n%s\n", getLogLevel(level), name, data)
	}
}

func WarningPrintf(format string, a ...interface{}) {
	level := JARVICE_WARNING_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func ErrorObj(name string, v interface{}) {
	level := JARVICE_ERROR_LOGGING
	if LogLevel() <= level {
		data, _ := json.MarshalIndent(v, "", " ")
		Log.Printf("%s %s:\n%s\n", getLogLevel(level), name, data)
	}
}

func ErrorPrintf(format string, a ...interface{}) {
	level := JARVICE_ERROR_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func CriticalObj(name string, v interface{}) {
	level := JARVICE_CRITICAL_LOGGING
	if LogLevel() <= level {
		data, _ := json.MarshalIndent(v, "", " ")
		Log.Printf("%s %s:\n%s\n", getLogLevel(level), name, data)
	}
}

func CriticalPrintf(format string, a ...interface{}) {
	level := JARVICE_CRITICAL_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}
