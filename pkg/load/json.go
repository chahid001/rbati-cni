package load

import (
	"encoding/json"
	"io"
	"os"
	"ft_cni/pkg/logs"
)

type IPAMConfig struct {
	Subnet 		string 			`json:"subnet"`
	Routes []struct {
		Dst 	string 			`json:"dst"`
	} 							`json:"routes"`
}

type CNIConfig struct {
	CNIVersion 	string        	`json:"cniVersion"`
	Name       	string        	`json:"name"`
	Type      	string     		`json:"type"`
	Bridge    	string     		`json:"bridge,omitempty"`
	BridgeIP	string			`json:"bridgeIP,omitempty"`
	IsGateway 	bool       		`json:"isGateway,omitempty"`
	IPAM      	*IPAMConfig		`json:"ipam,omitempty"`
}

func RawJson() *CNIConfig {

	data, err := io.ReadAll(os.Stdin)

	if err != nil {
		logs.Log.Info("Failed to read stdin: ", err.Error())
		os.Exit(1)
	}

	var config CNIConfig

	if err := json.Unmarshal(data, &config); err != nil {
		logs.Log.Info("Failed to Unmrashall")
		os.Exit(1)
	}

	return &config
}