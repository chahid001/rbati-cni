package result

import (
	"encoding/json"
	"fmt"
	"os"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/types"
)

func ReturnAdd(CNIResult *types.CNIResult) error {

	interfaceSlice := make([]map[string]string, 0, len(CNIResult.Interfaces))

	for _, iface := range CNIResult.Interfaces {

		ifaceMap := map[string]string{

			"name": iface.Name,
			"mac":  iface.MAC,
		}

		if iface.Sandbox != "" {
			ifaceMap["sandbox"] = iface.Sandbox
		}

		interfaceSlice = append(interfaceSlice, ifaceMap)
	}

	ipSlice := make([]map[string]interface{}, 0, len(CNIResult.IPs))

	for _, ip := range CNIResult.IPs {

		ipMAP := map[string]interface{}{

			"version":   ip.Version,
			"address":   ip.Address,
			"interface": ip.Interface,
		}

		ipSlice = append(ipSlice, ipMAP)

	}

	routeSlice := make([]map[string]string, 0, len(CNIResult.Routes))

	for _, route := range CNIResult.Routes {

		routeMap := map[string]string{

			"dst": route.Dst,
		}

		if route.GW != "" {
			routeMap["gw"] = route.GW
		}

		routeSlice = append(routeSlice, routeMap)

	}

	result := map[string]interface{}{
		"cniVersion": CNIResult.CNIVersion,
		"interfaces": interfaceSlice,
		"ips":        ipSlice,
		"routes":     routeSlice,
	}

	if CNIResult.DNS != nil {
		result["dns"] = map[string]interface{}{
			"nameservers": CNIResult.DNS.Nameservers,
			"search":      CNIResult.DNS.Search,
			"options":     CNIResult.DNS.Options,
			"domain":      CNIResult.DNS.Domain,
		}
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return err
	}

	logs.Log.Info("JSON Output", "JSON", string(output))

	fmt.Println(string(output))
	return nil
}
