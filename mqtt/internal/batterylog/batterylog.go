package batterylog

import (
	"log"
	"os"
)

var stdout = log.New(os.Stdout, "", log.Ldate|log.Ltime)

// Printf keeps battery diagnostics in the configured log file and exposes
// the same line through the container's standard output for live diagnosis.
func Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
	stdout.Printf(format, args...)
}

func Println(args ...interface{}) {
	log.Println(args...)
	stdout.Println(args...)
}

func Fatalf(format string, args ...interface{}) {
	Printf(format, args...)
	os.Exit(1)
}
