package load

import (
	"encoding/json"
	"io"
	"os"
	"ft_cni/pkg/logs"
)

type DNSConfig struct {
    Nameservers []string `json:"nameservers,omitempty"`
    Search      []string `json:"search,omitempty"`
    Options     []string `json:"options,omitempty"`
}

type IPAMConfig struct {
	Subnet 		string 			`json:"subnet"`
	// Address 	string 			`json:"address,omitempty"` for Later 
	Routes []struct {
		Dst 	string 			`json:"dst"`
	} 							`json:"routes"`
	DNS 		*DNSConfig		`json:"dns,omitempty"`
}

type CNIConfig struct {
	CNIVersion 	string        	`json:"cniVersion"`
	Name       	string        	`json:"name"`
	Type      	string     		`json:"type"`
	Bridge    	string     		`json:"bridge,omitempty"`
	BridgeIP	string			`json:"bridgeIP,omitempty"`
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