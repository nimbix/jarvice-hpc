package logger

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	Log *log.Logger = nil
)

func init() {
	logPath := "/tmp"
	if env := os.Getenv(LOG_PATH); len(env) > 0 {
		logPath = env
	} else if home := os.Getenv("HOME"); len(home) > 0 {
		logPath = home + "/.config/jarvice-hpc"
	}
	timeout := LOG_DEFAULT_TIMEOUT
	if env := os.Getenv(LOG_TIMEOUT); len(env) > 0 {
		if t, err := strconv.Atoi(env); err == nil {
			timeout = t
		}
	}
	// Create log directory if needed
	if err := os.MkdirAll(filepath.Clean(logPath), 0700); err != nil {
		log.Printf("logger cannot create directory: %v",
			fmt.Errorf("LogWriter: MkdirAll: %w", err))
		log.Printf("logging disable")
		return
	}
	logfile := filepath.Clean(logPath + "/jarvice-hpc.log")
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
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Printf("logger cannot open file: %v",
			fmt.Errorf("LogWriter: OpenFile: %w", err))
		log.Printf("logging disabled")
		return
	}
	if stat, serr := f.Stat(); serr == nil {
		if stat.Size() == 0 {
			f.WriteString(time.Now().Format(time.RFC3339) + "\n")
			f.Sync()
		}
	} else {
		log.Printf("logger cannot write to logfile: %v",
			fmt.Errorf("LogWriter: Stat: %w", serr))
		log.Printf("logging disabled")
		return
	}
	Log = log.New(f, "", log.LstdFlags)
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
	if Log == nil {
		return
	}
	level := JARVICE_DEBUG_LOGGING
	if LogLevel() <= level {
		Log.Printf("%s %s:\n%+v\n", getLogLevel(level), name, v)
	}
}

func DebugPrintf(format string, a ...interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_DEBUG_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func InfoObj(name string, v interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_INFO_LOGGING
	if LogLevel() <= level {
		Log.Printf("%s %s:\n%+v\n", getLogLevel(level), name, v)
	}
}

func InfoPrintf(format string, a ...interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_INFO_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func WarningObj(name string, v interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_WARNING_LOGGING
	if LogLevel() <= level {
		Log.Printf("%s %s:\n%+v\n", getLogLevel(level), name, v)
	}
}

func WarningPrintf(format string, a ...interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_WARNING_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func ErrorObj(name string, v interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_ERROR_LOGGING
	if LogLevel() <= level {
		Log.Printf("%s %s:\n%+v\n", getLogLevel(level), name, v)
	}
}

func ErrorPrintf(format string, a ...interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_ERROR_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}

func CriticalObj(name string, v interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_CRITICAL_LOGGING
	if LogLevel() <= level {
		Log.Printf("%s %s:\n%+v\n", getLogLevel(level), name, v)
	}
}

func CriticalPrintf(format string, a ...interface{}) {
	if Log == nil {
		return
	}
	level := JARVICE_CRITICAL_LOGGING
	if LogLevel() <= level {
		prefix := getLogLevel(level) + " "
		Log.Printf(prefix+format, a...)
	}
}
