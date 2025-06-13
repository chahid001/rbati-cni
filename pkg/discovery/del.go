package discovery

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"net"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)


func getPodIP(NetnsPath, PodVethEndName string) (net.IP, error) {

	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logs.Log.Error("Couldn't Open Pod Namespace.", "ERROR", err)
		return nil, err
	}
	defer podNS.Close()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	
	origNS, err := netns.Get()
	if err != nil {
		logs.Log.Error("Couldn't Save the current namespace.", "ERROR", err)
		return nil, err
	}
	defer origNS.Close()
	
	if err := netns.Set(podNS); err != nil {
		logs.Log.Error("Couldn't switch current context to POD namespace.", "ERROR", err)
		return nil, err
	}
	
	defer func() {
		netns.Set(origNS)
		logs.Log.Info("Switched back to Host namespace")
	} ()
	
	veth, err := netlink.LinkByName(PodVethEndName)
	if err != nil {
		logs.Log.Error("Couldn't find POD veth end.", "ERROR", err)
		return nil, err
	
	}

    addrs, err := netlink.AddrList(veth, netlink.FAMILY_V4)
    if err != nil || len(addrs) == 0 {
        logs.Log.Error("Couldn't get IP for POD veth end", "ERROR", err)
        return nil, err
    }
    podIP := addrs[0].IP 

	return podIP, nil
}

func DEL(envMap map[string]string, config *load.CNIConfig) error {

	logs.Log.Info("----------------------------")
	logs.Log.Info("----------------------------")
	logs.Log.Info("Begin ADD operation for", "POD", envMap["CNI_CONTAINERID"][:4])

	podIP, err := getPodIP(envMap["CNI_NETNS"], envMap["CNI_IFNAME"])

	if err != nil {
		logs.Log.Error("Couldn't Delete & Release POD IPv4.", "ERROR", err)
		return err
	}



	if err != nil {
		logs.Log.Error("Couldn't Load Subnet for IPv4 Deletion.", "ERROR", err)
		return err		
	}

	container_id := envMap["CNI_CONTAINERID"][:4]
	vethHostName := "veth" + container_id

	vethHostEnd, err := netlink.LinkByName(vethHostName)

	if err != nil {
		logs.Log.Error("Couldn't Find Veth Host end.", "ERROR", err)
		return err		
	}

	if err := netlink.LinkDel(vethHostEnd); err != nil {
		logs.Log.Error("Couldn't Delete Veth Host end.", "ERROR", err)
		return err				
	}

	i, err := network.NewIPAM(config.IPAM.Subnet)
	logs.Log.Info("Releasing IP", "IP", podIP.String())
	
	if err := i.Release(&podIP); err != nil {
		logs.Log.Error("Couldn't release IP Address", "ERROR", err)	
		return err	
	}
	return nil
}