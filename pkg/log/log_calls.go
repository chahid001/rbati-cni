package log 

import (
	"os"
	"fmt"
)
func LogCalls(envMap map[string]string, jsonData string) {

	f, err := os.OpenFile("/tmp/cni-input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

 	if err != nil {
 		os.Stderr.WriteString("Failed to open log file: " + err.Error() + "\n")
 		os.Exit(1)
 	}
 	defer f.Close()

 	logEntry := ""

	for key, value := range envMap {
		logEntry += fmt.Sprintf("%s: %s\n", key, value)
	}
	logEntry += fmt.Sprintf("%s\n\n", jsonData)

 	if _, err := f.Write([]byte(logEntry)); err != nil {
 		os.Stderr.WriteString("Failed to write to log file: " + err.Error() + "\n")
 		os.Exit(1)
 	}

}