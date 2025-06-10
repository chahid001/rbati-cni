package main

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"ft_cni/pkg/discovery"
	"os"
)



var (
	envMap 	map[string]string
	configJSON *load.CNIConfig
)

func init() {
	logs.Init("/tmp/mycni.log")
	logs.Log.Info("Init Function | Initialized Log")
	envMap = load.ENV()
	logs.Log.Info("env check")
	configJSON = load.RawJson()
	logs.Log.Info("Logjson check")
	network.EnableNat()
}

func main() {

	cmd := envMap["CNI_COMMAND"]
	
	switch cmd {		

	case "ADD":
		if err := discovery.Add(envMap, configJSON); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	
	case "DEL":
		if err := discovery.DEL(envMap, configJSON); err != nil {
			os.Exit(1)
		}
		os.Exit(0)

	// case "CHECK":
	case "VERSION":

	}

}


