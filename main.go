package main

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/targets"
	"os"
)



var (
	envMap map[string]string
	rawJSON []byte
	
)

func init() {

	envMap = load.ENV()
	rawJSON = load.RawJson()


}



func main() {

	cmd := envMap["CNI_COMMAND"]
	
	switch cmd {

		

	case "ADD":
		targets.Add(envMap)
		os.Exit(0)
	}
}


