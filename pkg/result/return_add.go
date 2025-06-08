package result

import (
	"fmt"
	"encoding/json"
	"os"

)


func ReturnAdd(SandBox, Address, Gateway string) {

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
		"dns": map[string]string{},
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}