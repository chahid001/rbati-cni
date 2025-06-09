package result

import (
	"encoding/json"
	"fmt"
	"ft_cni/pkg/load"
	"os"
)


func ReturnAdd(SandBox, Address, Gateway string, DNS *load.DNSConfig) {

	result := map[string]interface{}{
		"cniVersion": "0.4.0",
		"interfaces": []map[string]string{
			{
				"name":    "eth0",
				"sandbox": SandBox,
			},
		},
		"ips": []map[string]string{
			{
				"version": "4",
				"address": Address,
				"gateway": Gateway,
			},
		},
		"dns": map[string]interface{}{
			
			"nameservers": DNS.Nameservers,
			"search": DNS.Search,
			"options": DNS.Options,
			
		},
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}