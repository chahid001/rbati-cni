package main

import (
	"ft_cni/pkg/discovery"
	"ft_cni/pkg/load"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"ft_cni/pkg/result"
	"ft_cni/pkg/types"
	"net"
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
	network.SetupVXLAN("vxlan0", 1001, net.ParseIP("192.168.56.111"), net.ParseIP("192.168.56.110"))
}

func main() {

	cmd := envMap["CNI_COMMAND"]
	
	if cmd == "ADD" || cmd == "DEL" || cmd == "CHECK" {
		if discovery.CheckVersions(configJSON.CNIVersion) == false {
			
			cniErr := types.CNIError {
				CNIVersion: configJSON.CNIVersion,
				Code: 4,
				Msg: "Incompatible CNI version",
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)			
		}
	}

	switch cmd {		

	case "ADD":
		if err := discovery.Add(envMap, configJSON); err != nil {
			
			cniErr := types.CNIError {
				CNIVersion: configJSON.CNIVersion,
				Code: 100,
				Msg: "Failed in ADD operation.",
				Details: err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)
	
	case "DEL":
		if err := discovery.DEL(envMap, configJSON); err != nil {
			
			cniErr := types.CNIError {
				CNIVersion: configJSON.CNIVersion,
				Code: 100,
				Msg: "Failed in DEL operation.",
				Details: err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)

	case "CHECK":
		if err := discovery.CHECK(envMap, configJSON); err != nil {
			
			cniErr := types.CNIError {
				CNIVersion: configJSON.CNIVersion,
				Code: 100,
				Msg: "Failed in CHECK operation.",
				Details: err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)	

	case "VERSION":

		if err := discovery.VERSION(); err != nil {
			
			cniErr := types.CNIError {
				CNIVersion: configJSON.CNIVersion,
				Code: 100,
				Msg: "Failed in VERSION operation.",
				Details: err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)		

	}

}


