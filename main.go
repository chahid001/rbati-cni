package main

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/targets"
	"ft_cni/pkg/network"
	"os"
)



var (
	envMap map[string]string
	rawJSON []byte
	
)

func init() {

	envMap = load.ENV()
	rawJSON = load.RawJson()

	network.EnableNat()
}



func main() {

	cmd := envMap["CNI_COMMAND"]
	
	switch cmd {

		

	case "ADD":
		targets.Add(envMap)
		os.Exit(0)
	}
}


