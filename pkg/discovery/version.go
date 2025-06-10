package discovery

import (
	"encoding/json"
	"fmt"
	"ft_cni/pkg/logs"
)


var supported_versions = []string{"0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0"}


func CheckVersions(Version string) bool {

	for _, sup_v := range supported_versions {

		if Version == sup_v {
			return true 
		}

	}

	return false

}

func VERSION() error {

	versions := map[string]interface{}{
        "cniVersion":        "0.4.0",
        "supportedVersions": supported_versions,
    }

    output, err := json.MarshalIndent(versions, "", "  ")
    if err != nil {
        logs.Log.Error("Failed to encode version info.")
		return err
    }

    fmt.Println(string(output))

	return nil
}