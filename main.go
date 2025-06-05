package main

import (
	"fmt"
	"ft_cni/pkg/load"
	// "io"
	"os"
)

var (
	envMap map[string]string
	// rawJSON := []byte
)

func init() {

	envMap = load.ENV()

}

// 	data, err := io.ReadAll(os.Stdin)
// 	if err != nil {
// 		os.Stderr.WriteString("Failed to read stdin: " + err.Error() + "\n")
// 		os.Exit(1)
// 	}

func main() {

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
	logEntry += fmt.Sprintf("\n")

 	if _, err := f.Write([]byte(logEntry)); err != nil {
 		os.Stderr.WriteString("Failed to write to log file: " + err.Error() + "\n")
 		os.Exit(1)
 	}
}


