package main

import (
	"os"
	"path/filepath"
	"rbati-cni/internal/discovery"
	"rbati-cni/internal/ipam"
	"rbati-cni/internal/load"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/types"
	"rbati-cni/internal/utils"
	"strings"

	"github.com/vishvananda/netlink"
)

var (
	envMap     map[string]string
	configJSON *load.CNIConfig
)


func init() {
	logs.Init()
	envMap = load.ENV()
	configJSON = load.RawJSONFromStdin()
}

func main() {

	bridge := types.BRIDGE_NAME

	if configJSON.Bridge != "" {
		bridge = configJSON.Bridge
	}

	_, err := netlink.LinkByName(bridge)
    if err != nil {
        utils.CreateErr(configJSON.CNIVersion, "Network not ready", "CNI bridge device not found", 100)
        os.Exit(1)
    }

	cmd := envMap["CNI_COMMAND"]

	env_path := filepath.Join(types.ENV_PATH, types.ENV_FILENAME)
	data, err := os.ReadFile(env_path)
	if err != nil {
		logs.Log.Error("Failed to read env.", "ERROR", err)
		os.Exit(1)
	}
	podSubnet := strings.TrimSpace(string(data))

	ipam_path := filepath.Join(types.IPAM_FODLER_PATH, types.IPAM_DB_FILENAME)
	ipam, err := ipam.NewIPAM(podSubnet, ipam_path)
	if err != nil {
		logs.Log.Error("Failed create ipam object.", "ERROR", err)
		os.Exit(1)
	}

	if cmd == "ADD" || cmd == "DEL" || cmd == "CHECK" {
		if discovery.CheckVersions(configJSON.CNIVersion) == false {

			utils.CreateErr(configJSON.CNIVersion, "Incompatible CNI version", "", 1)
		}
	}

	switch cmd {

	case "ADD":
		if err := discovery.Add(envMap, configJSON, bridge, ipam); err != nil {

			utils.CreateErr(configJSON.CNIVersion, "Failed in ADD operation.", err.Error(), 100)

		}
		os.Exit(0)

	case "DEL":
		if err := discovery.DEL(envMap, configJSON, ipam); err != nil {

			utils.CreateErr(configJSON.CNIVersion, "Failed in DEL operation.", err.Error(), 100)
		}
		os.Exit(0)

	case "CHECK":
		if err := discovery.CHECK(envMap, configJSON, podSubnet); err != nil {

			utils.CreateErr(configJSON.CNIVersion, "Failed in CHECK operation.", err.Error(), 100)
		}
		os.Exit(0)

	case "VERSION":

		if err := discovery.VERSION(); err != nil {
			utils.CreateErr(configJSON.CNIVersion, "Failed in VERSION operation.", err.Error(), 100)
		}
		os.Exit(0)

	}

}