package load

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"rbati-cni/internal/logs"
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
	Routes []*RoutesConfig `json:"routes"`
	DNS    *DNSConfig      `json:"dns,omitempty"`
}

type CNIConfig struct {
	CNIVersion 			string      `json:"cniVersion"`
	Name       			string      `json:"name"`
	Type       			string      `json:"type"`
	Bridge     			string      `json:"bridge,omitempty"`
	IFace      			string      `json:"iface"`
	MetricsPort			string		`json:"metrics_port"`
	IPAM       			*IPAMConfig `json:"ipam,omitempty"`
}

// For CNI plugin (stdin use)
func RawJSONFromStdin() *CNIConfig {
	return parseCNIConfig(os.Stdin)
}

// For controller (file use)
func LoadFromFile(path string) *CNIConfig {
	file, err := os.Open(path)
	if err != nil {
		log.Println("[ERROR] Couldn't open the config file !")
		os.Exit(1)
	}
	defer file.Close()
	return parseCNIConfig(file)
}

// Shared logic
func parseCNIConfig(r io.Reader) *CNIConfig {
	data, err := io.ReadAll(r)
	if err != nil {
		logs.Log.Error("Failed to read config", "ERROR", err)
		os.Exit(1)
	}

	var config CNIConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logs.Log.Error("Failed to unmarshal config", "ERROR", err)
		os.Exit(1)
	}

	return &config
}
