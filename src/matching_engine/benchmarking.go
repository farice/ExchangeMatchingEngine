package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var benchmarkLogFile *os.File

func CreateBenchmarkingLog() {

	file, err := os.OpenFile("/var/log/erss/benchmarks.csv", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		benchmarkLogFile = file
	} else {
		log.Info("Failed to log to file, using default stdout")
	}
	log.Info("Benchmark logging initialized.")
}

// Write time elapsed for method to the main log output.
func LogMethodTimeElapsed(methodName string, start time.Time) {
	elapsed := time.Since(start)
	log.Printf(">> %s took %s", methodName, elapsed)
	benchmarkLogFile.WriteString(fmt.Sprintf("%s, %s\n", methodName, elapsed))
}
