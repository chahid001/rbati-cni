package logger

import (
	"os"
	"log"
)


var (
	Log *log.Logger
)

func Init(logFile string) error {
	file, err := os.OpenFile("/tmp/cni.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	Log = log.New(file, "CNI: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

// var logging *log.Logger

// func LogCalls(envMap map[string]string, jsonData string) {

	
// 	f, err := os.OpenFile("/tmp/cni-input.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

//  	if err != nil {
//  		os.Stderr.WriteString("Failed to open log file: " + err.Error() + "\n")
//  		os.Exit(1)
//  	}
//  	defer f.Close()

//  	logEntry := ""

// 	for key, value := range envMap {
// 		logEntry += fmt.Sprintf("%s: %s\n", key, value)
// 	}
// 	logEntry += fmt.Sprintf("%s\n\n", jsonData)

//  	if _, err := f.Write([]byte(logEntry)); err != nil {
//  		os.Stderr.WriteString("Failed to write to log file: " + err.Error() + "\n")
//  		os.Exit(1)
//  	}

// 	logging = log.New(f, "", log.LstdFlags)
// }