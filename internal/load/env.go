package load

import "os"


func ENV() map[string]string {

	envNames := []string { // based on CNI spec at https://github.com/containernetworking/cni/blob/main/SPEC.md
		"CNI_COMMAND",
		"CNI_CONTAINERID",
		"CNI_NETNS",
		"CNI_IFNAME",
		"CNI_ARGS",
		"CNI_PATH",
	}

	envMap := make(map[string]string, len(envNames))

	for _, name := range envNames { // _ for ignoring the index

		envMap[name] = os.Getenv(name)

	}

	return envMap
}