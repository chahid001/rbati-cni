package load

import (
	"encoding/json"
	"ft_cni/internal/logs"
	"io"
	"os"
)

type DNSConfig struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	Search      []string `json:"search,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type RoutesConfig struct {
	Dst string `json:"dst"`
	Gw  string `json:"gw"`
}

type IPAMConfig struct {
	Subnet string          `json:"subnet"`
	Routes []*RoutesConfig `json:"routes"`
	DNS    *DNSConfig      `json:"dns,omitempty"`
}

type CNIConfig struct {
	CNIVersion string      `json:"cniVersion"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Bridge     string      `json:"bridge,omitempty"`
	BridgeIP   string      `json:"bridgeIP,omitempty"`
	IFace      string      `json:"iface"`
	IPAM       *IPAMConfig `json:"ipam,omitempty"`
}

func RawJson() *CNIConfig {

	data, err := io.ReadAll(os.Stdin)

	// logs.Log.Error("Stdin", "JSON", string(data)) for Debug

	if err != nil {
		logs.Log.Error("Failed to read stdin", "ERROR", err)
		os.Exit(1)
	}

	var config CNIConfig

	if err := json.Unmarshal(data, &config); err != nil {
		logs.Log.Error("Failed to Unmrashall", "ERROR", err)
		os.Exit(1)
	}

	return &config
}
