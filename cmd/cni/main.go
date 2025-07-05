package main

import (
	"os"
	"rbati-cni/internal/discovery"
	"rbati-cni/internal/ipam"
	"rbati-cni/internal/load"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/result"
	"rbati-cni/internal/types"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

var (
	envMap     map[string]string
	configJSON *load.CNIConfig
)

const (
	vxlanName 	= "rbati.1"
	bridgeName	= "rbati.0"
)

func init() {
	logs.Init("/tmp/mycni.log")
	logs.Log.Info("Init Function | Initialized Log")
	envMap = load.ENV()
	logs.Log.Info("env check")
	configJSON = load.RawJSONFromStdin()
	logs.Log.Info("Logjson check")
}

func main() {

	bridge := bridgeName

	if configJSON.Bridge != "" {
		bridge = configJSON.Bridge
	}

	for { // Waiting for Bridge and Vxlan interfaces to be created
		_, errBR := netlink.LinkByName(bridge)
		_, errVX := netlink.LinkByName(vxlanName)
		if errBR == nil && errVX == nil {
			break
		}
		logs.Log.Info("Waiting for CNI Agent to be initialized...")
		time.Sleep(2 * time.Second)
	}

	cmd := envMap["CNI_COMMAND"]

	data, err := os.ReadFile("/run/rbati-cni/rbati.env")
	if err != nil {
		logs.Log.Error("Failed to read env.", "ERROR", err)
		os.Exit(1)
	}
	podSubnet := strings.TrimSpace(string(data))

	ipam, err := ipam.NewIPAM(podSubnet, "/var/lib/cni/rbati-cni/ipam.db")
	if err != nil {
		logs.Log.Error("Failed create ipam object.", "ERROR", err)
		os.Exit(1)
	}

	if cmd == "ADD" || cmd == "DEL" || cmd == "CHECK" {
		if discovery.CheckVersions(configJSON.CNIVersion) == false {

			cniErr := types.CNIError{
				CNIVersion: configJSON.CNIVersion,
				Code:       4,
				Msg:        "Incompatible CNI version",
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
	}

	switch cmd {

	case "ADD":
		if err := discovery.Add(envMap, configJSON, bridge, ipam); err != nil {

			cniErr := types.CNIError{
				CNIVersion: configJSON.CNIVersion,
				Code:       100,
				Msg:        "Failed in ADD operation.",
				Details:    err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)

	case "DEL":
		if err := discovery.DEL(envMap, configJSON, ipam); err != nil {

			cniErr := types.CNIError{
				CNIVersion: configJSON.CNIVersion,
				Code:       100,
				Msg:        "Failed in DEL operation.",
				Details:    err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)

	case "CHECK":
		if err := discovery.CHECK(envMap, configJSON, podSubnet); err != nil {

			cniErr := types.CNIError{
				CNIVersion: configJSON.CNIVersion,
				Code:       100,
				Msg:        "Failed in CHECK operation.",
				Details:    err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)

	case "VERSION":

		if err := discovery.VERSION(); err != nil {

			cniErr := types.CNIError{
				CNIVersion: configJSON.CNIVersion,
				Code:       100,
				Msg:        "Failed in VERSION operation.",
				Details:    err.Error(),
			}

			result.ReturnErr(&cniErr)

			os.Exit(1)
		}
		os.Exit(0)

	}

}