package discovery

import (
	"net"
	"rbati-cni/internal/load"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/utils"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func CHECK(envMap map[string]string, config *load.CNIConfig) error {

	logs.Log.Info("----------------------------")
	logs.Log.Info("----------------------------")
	logs.Log.Info("Begin CHECK operation for", "POD", envMap["CNI_CONTAINERID"][:4])

	// Check Nampespace
	podNS, err := netns.GetFromPath(envMap["CNI_NETNS"])
	if err != nil {
		logs.Log.Error("Couldn't Open Pod Namespace.", "ERROR", err)
		return err
	}
	defer podNS.Close()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNS, err := netns.Get()
	if err != nil {
		logs.Log.Error("Couldn't Save the current namespace.", "ERROR", err)
		return err
	}
	defer origNS.Close()

	if err := netns.Set(podNS); err != nil {
		logs.Log.Error("Couldn't switch current context to POD namespace.", "ERROR", err)
		return err
	}

	defer func() {
		netns.Set(origNS)
		logs.Log.Info("Switched back to Host namespace")
	}()

	// Check Interface
	_, err = netlink.LinkByName(envMap["CNI_IFNAME"])

	if err != nil {
		logs.Log.Error("POD Interface not found", "ERROR", err)
		return err
	}

	// Check IPs
	Addrs, err := utils.GetPodIP(envMap["CNI_NETNS"], envMap["CNI_IFNAME"])

	_, subnet, err := net.ParseCIDR(config.IPAM.Subnet)

	if err != nil {
		logs.Log.Error("Couldn't parse Subnet", "ERROR", err)
		return err
	}

	for _, Address := range Addrs {

		if !subnet.Contains(Address.IP) {
			logs.Log.Error("IP does not belong to Subent", "IP", Address.IP.String())
			return err
		}
	}

	return nil
}
